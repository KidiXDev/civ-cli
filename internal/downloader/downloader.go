package downloader

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// Tuning constants – tweak for different network conditions
// ---------------------------------------------------------------------------

const (
	defaultChunks     = 8                // parallel connections
	defaultMinChunk   = 2 * 1024 * 1024  // 2 MB minimum per chunk
	defaultBufferSize = 256 * 1024       // 256 KB read buffer
	minParallelSize   = 10 * 1024 * 1024 // 10 MB – below this use single-stream
	maxChunks         = 32               // hard upper limit
	probeTimeout      = 15 * time.Second // timeout for HEAD/Range probe
)

// ---------------------------------------------------------------------------
// Public option / result types
// ---------------------------------------------------------------------------

// DownloadOptions configures a single download operation.
type DownloadOptions struct {
	OutputDir       string           // destination directory (falls back to config)
	Filename        string           // override auto-detected filename
	Chunks          int              // parallel chunk count (0 = auto)
	MaxRetries      int              // per-chunk retry count (0 = use config)
	ProgressCb      ProgressCallback // real-time progress updates
	MinChunkSize    int64            // min bytes per chunk to enable parallel
	BufferSize      int              // per-read buffer size in bytes
	ForceSequential bool             // disable parallel download
}

// DownloadResult contains metrics about a completed download.
type DownloadResult struct {
	FilePath   string
	FileSize   int64
	Duration   time.Duration
	AvgSpeed   float64 // bytes/sec
	ChunksUsed int
}

// ---------------------------------------------------------------------------
// Internal probe result
// ---------------------------------------------------------------------------

type fileProbe struct {
	ContentLength int64
	AcceptRanges  bool
	Filename      string
	FinalURL      string // URL after redirects (may be a signed CDN link)
}

// ---------------------------------------------------------------------------
// Downloader
// ---------------------------------------------------------------------------

// Downloader manages file downloads with parallel chunking, retry with
// exponential back-off, and real-time progress tracking.
type Downloader struct {
	transport  *http.Client
	cfg        *config.Config
	chunks     int
	bufferSize int
}

// NewDownloader creates a Downloader with an aggressively-tuned HTTP transport
// for maximum throughput.
func NewDownloader(cfg *config.Config) *Downloader {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   maxChunks, // at least as many as we'll use
		MaxConnsPerHost:       0,         // unlimited
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		WriteBufferSize:       defaultBufferSize,
		ReadBufferSize:        defaultBufferSize,
		DisableCompression:    true, // model files are already compressed
		ForceAttemptHTTP2:     true,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	numChunks := defaultChunks
	if n := runtime.NumCPU(); n > numChunks {
		numChunks = n
	}
	if numChunks > maxChunks {
		numChunks = maxChunks
	}

	return &Downloader{
		transport: &http.Client{
			Transport: t,
			Timeout:   0, // no global timeout – each operation has its own
		},
		cfg:        cfg,
		chunks:     numChunks,
		bufferSize: defaultBufferSize,
	}
}

// Close releases idle connections held by the transport pool.
func (d *Downloader) Close() {
	d.transport.CloseIdleConnections()
}

// ---------------------------------------------------------------------------
// Primary API – Download
// ---------------------------------------------------------------------------

// Download performs a high-speed download of a Civitai model version.
//
// Strategy:
//  1. Probe the URL (HEAD → Range GET fallback) to discover file size,
//     Range support, filename, and the final CDN URL.
//  2. If the server supports HTTP Range requests and the file exceeds
//     minParallelSize, split it into N chunks and download in parallel
//     using direct WriteAt (no merge step).
//  3. Each chunk retries independently with exponential back-off.
//  4. Progress is reported through opts.ProgressCb.
func (d *Downloader) Download(ctx context.Context, versionID int, opts DownloadOptions) (*DownloadResult, error) {
	start := time.Now()

	downloadURL := fmt.Sprintf("https://civitai.com/api/download/models/%d", versionID)
	if d.cfg.APIKey != "" {
		u, err := url.Parse(downloadURL)
		if err != nil {
			return nil, fmt.Errorf("invalid download URL: %w", err)
		}
		q := u.Query()
		q.Set("token", d.cfg.APIKey)
		u.RawQuery = q.Encode()
		downloadURL = u.String()
	}

	// Phase 1 — probe ---------------------------------------------------------
	log.Debug().Str("url", downloadURL).Msg("probing download URL")
	probe, err := d.probeURL(ctx, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to probe download: %w", err)
	}
	log.Info().
		Int64("size", probe.ContentLength).
		Bool("ranges", probe.AcceptRanges).
		Str("file", probe.Filename).
		Msg("probe complete")

	// Resolve filename --------------------------------------------------------
	filename := opts.Filename
	if filename == "" {
		filename = probe.Filename
	}
	if filename == "" {
		filename = fmt.Sprintf("model_%d.safetensors", versionID)
	}

	// Resolve output directory ------------------------------------------------
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = d.cfg.DefaultDownloadDir
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}
	outPath := filepath.Join(outputDir, filename)

	// Decide strategy ---------------------------------------------------------
	numChunks := opts.Chunks
	if numChunks <= 0 {
		numChunks = d.chunks
	}
	minChunk := opts.MinChunkSize
	if minChunk <= 0 {
		minChunk = defaultMinChunk
	}
	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = d.bufferSize
	}

	useParallel := probe.AcceptRanges &&
		probe.ContentLength > int64(minParallelSize) &&
		!opts.ForceSequential &&
		numChunks > 1

	// Ensure each chunk is at least minChunk bytes
	if useParallel && probe.ContentLength/int64(numChunks) < minChunk {
		numChunks = int(probe.ContentLength / minChunk)
		if numChunks < 1 {
			numChunks = 1
		}
	}

	actualChunks := 1
	if useParallel {
		actualChunks = numChunks
	}

	tracker := newProgressTracker(probe.ContentLength, actualChunks, opts.ProgressCb)

	// Phase 2 — download ------------------------------------------------------
	log.Info().
		Bool("parallel", useParallel).
		Int("chunks", actualChunks).
		Int("buffer_kb", bufSize/1024).
		Str("file", filename).
		Msg("starting download")

	if useParallel {
		err = d.downloadParallel(ctx, probe, outPath, actualChunks, bufSize, tracker)
	} else {
		err = d.downloadSequential(ctx, probe, outPath, bufSize, tracker)
	}
	if err != nil {
		os.Remove(outPath) // clean partial file
		return nil, err
	}

	tracker.notifyComplete()

	elapsed := time.Since(start)
	var avgSpeed float64
	if elapsed.Seconds() > 0 {
		avgSpeed = float64(probe.ContentLength) / elapsed.Seconds()
	}

	log.Info().
		Str("path", outPath).
		Str("speed", FormatSpeed(avgSpeed)).
		Dur("elapsed", elapsed).
		Msg("download complete")

	return &DownloadResult{
		FilePath:   outPath,
		FileSize:   probe.ContentLength,
		Duration:   elapsed,
		AvgSpeed:   avgSpeed,
		ChunksUsed: actualChunks,
	}, nil
}

// ---------------------------------------------------------------------------
// Backward-compatible writer-based API
// ---------------------------------------------------------------------------

// DownloadToWriter is a convenience wrapper that bridges the new parallel
// download engine with a legacy io.Writer-based progress interface.
// It is retained for backward compatibility with CLI progress bars.
func (d *Downloader) DownloadToWriter(ctx context.Context, versionID int, outputDir string, progressWriter io.Writer) (string, error) {
	var cb ProgressCallback
	if progressWriter != nil {
		var lastReported int64
		reportBuf := make([]byte, 32*1024) // reusable scratch buffer
		var maxSet bool

		cb = func(p Progress) {
			// Set the max on the first callback (for progressbar-style writers)
			if !maxSet {
				if s, ok := progressWriter.(interface{ ChangeMax64(int64) }); ok {
					s.ChangeMax64(p.TotalBytes)
				}
				maxSet = true
			}
			delta := p.DownloadedBytes - lastReported
			if delta <= 0 {
				return
			}
			// Write incremental byte counts to keep the writer's counter accurate
			for delta > 0 {
				n := delta
				if n > int64(len(reportBuf)) {
					n = int64(len(reportBuf))
				}
				progressWriter.Write(reportBuf[:n])
				delta -= n
			}
			lastReported = p.DownloadedBytes
		}
	}

	result, err := d.Download(ctx, versionID, DownloadOptions{
		OutputDir:  outputDir,
		ProgressCb: cb,
	})
	if err != nil {
		return "", err
	}
	return result.FilePath, nil
}

// ---------------------------------------------------------------------------
// URL probing
// ---------------------------------------------------------------------------

// probeURL discovers file size, Range support, filename, and the final URL
// (after CDN redirects). It tries HEAD first, then falls back to a tiny
// Range GET.
func (d *Downloader) probeURL(ctx context.Context, rawURL string) (*fileProbe, error) {
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	// --- attempt 1: HEAD ---
	req, err := http.NewRequestWithContext(probeCtx, http.MethodHead, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.transport.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent) && resp.ContentLength > 0 {
			return &fileProbe{
				ContentLength: resp.ContentLength,
				AcceptRanges:  resp.Header.Get("Accept-Ranges") == "bytes",
				Filename:      extractFilename(resp.Header.Get("Content-Disposition")),
				FinalURL:      resp.Request.URL.String(),
			}, nil
		}
	}

	// --- attempt 2: Range GET bytes=0-0 ---
	probeCtx2, cancel2 := context.WithTimeout(ctx, probeTimeout)
	defer cancel2()

	req, err = http.NewRequestWithContext(probeCtx2, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err = d.transport.Do(req)
	if err != nil {
		return nil, fmt.Errorf("probe failed: %w", err)
	}
	defer func() {
		_, _ = io.CopyN(io.Discard, resp.Body, 1024)
		_ = resp.Body.Close()
	}()

	probe := &fileProbe{
		FinalURL: resp.Request.URL.String(),
		Filename: extractFilename(resp.Header.Get("Content-Disposition")),
	}

	switch resp.StatusCode {
	case http.StatusPartialContent:
		probe.AcceptRanges = true
		probe.ContentLength = parseContentRange(resp.Header.Get("Content-Range"))
	case http.StatusOK:
		probe.ContentLength = resp.ContentLength
		probe.AcceptRanges = resp.Header.Get("Accept-Ranges") == "bytes"
	default:
		return nil, fmt.Errorf("unexpected probe status %d", resp.StatusCode)
	}

	return probe, nil
}

// ---------------------------------------------------------------------------
// Parallel download
// ---------------------------------------------------------------------------

func (d *Downloader) downloadParallel(ctx context.Context, probe *fileProbe, outPath string, numChunks int, bufSize int, tracker *progressTracker) error {
	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Pre-allocate the file to the full size
	if err := file.Truncate(probe.ContentLength); err != nil {
		return fmt.Errorf("preallocate: %w", err)
	}

	// Build chunks
	chunkSize := probe.ContentLength / int64(numChunks)
	chunks := make([]chunk, numChunks)
	for i := range numChunks {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == numChunks-1 {
			end = probe.ContentLength - 1 // last chunk takes the remainder
		}
		chunks[i] = chunk{id: i, start: start, end: end}
		tracker.initChunk(i, start, end)
	}

	// Launch workers
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, numChunks)
	var wg sync.WaitGroup

	for _, c := range chunks {
		wg.Add(1)
		go func(c chunk) {
			defer wg.Done()
			if err := d.downloadChunk(ctx, probe.FinalURL, c, file, tracker); err != nil {
				cancel() // abort remaining chunks
				errCh <- err
			}
		}(c)
	}

	wg.Wait()
	close(errCh)

	// Return the first error
	for e := range errCh {
		return e
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sequential download (fallback)
// ---------------------------------------------------------------------------

func (d *Downloader) downloadSequential(ctx context.Context, probe *fileProbe, outPath string, bufSize int, tracker *progressTracker) error {
	maxRetries := d.cfg.RetryCount
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			log.Debug().Int("attempt", attempt).Dur("backoff", backoff).Msg("retrying sequential download")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		lastErr = d.doSequentialDownload(ctx, probe.FinalURL, outPath, bufSize, tracker)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Warn().Err(lastErr).Int("attempt", attempt).Msg("sequential download failed")
	}
	return lastErr
}

func (d *Downloader) doSequentialDownload(ctx context.Context, url string, outPath string, bufSize int, tracker *progressTracker) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := d.transport.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	tracker.initChunk(0, 0, tracker.totalBytes)
	tracker.setChunkState(0, ChunkDownloading)

	buf := make([]byte, bufSize)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			tracker.addBytes(int64(n))
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	tracker.setChunkState(0, ChunkComplete)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// extractFilename parses the filename from a Content-Disposition header.
func extractFilename(cd string) string {
	if cd == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}
	filename := params["filename"]
	return strings.Trim(filename, "\"")
}
