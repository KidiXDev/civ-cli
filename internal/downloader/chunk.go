package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// chunk describes a byte range to download independently.
type chunk struct {
	id    int
	start int64
	end   int64 // inclusive
}

// size returns the number of bytes covered by this chunk.
func (c chunk) size() int64 { return c.end - c.start + 1 }

// ---------------------------------------------------------------------------
// Chunk download with retry + exponential back-off
// ---------------------------------------------------------------------------

func (d *Downloader) downloadChunk(ctx context.Context, url string, c chunk, file *os.File, tracker *progressTracker) error {
	maxRetries := d.cfg.RetryCount
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			tracker.setChunkState(c.id, ChunkRetrying)
			tracker.incChunkRetries(c.id)

			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			log.Debug().
				Int("chunk", c.id).
				Int("attempt", attempt).
				Dur("backoff", backoff).
				Msg("retrying chunk")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		lastErr = d.doChunkDownload(ctx, url, c, file, tracker)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Warn().Err(lastErr).Int("chunk", c.id).Int("attempt", attempt).Msg("chunk download failed")
	}

	tracker.setChunkState(c.id, ChunkFailed)
	return fmt.Errorf("chunk %d failed after %d retries: %w", c.id, maxRetries, lastErr)
}

// doChunkDownload performs a single HTTP Range GET for the given chunk.
// It writes directly into `file` at the correct offset using WriteAt
// (thread-safe for non-overlapping regions).
func (d *Downloader) doChunkDownload(ctx context.Context, url string, c chunk, file *os.File, tracker *progressTracker) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", c.start, c.end))

	resp, err := d.transport.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("status %d for chunk %d: %s", resp.StatusCode, c.id, string(body))
	}

	tracker.setChunkState(c.id, ChunkDownloading)

	buf := make([]byte, d.bufferSize)
	offset := c.start
	var chunkDownloaded int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.WriteAt(buf[:n], offset); writeErr != nil {
				return fmt.Errorf("write error at offset %d: %w", offset, writeErr)
			}
			offset += int64(n)
			chunkDownloaded += int64(n)
			tracker.addBytes(int64(n))
			tracker.setChunkDownloaded(c.id, chunkDownloaded)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	tracker.setChunkState(c.id, ChunkComplete)
	log.Debug().
		Int("chunk", c.id).
		Int64("bytes", chunkDownloaded).
		Msg("chunk download complete")
	return nil
}

// ---------------------------------------------------------------------------
// Content-Range header parser
// ---------------------------------------------------------------------------

// parseContentRange extracts the total size from a Content-Range header.
// Format: "bytes 0-0/12345678" → 12345678.  Returns -1 on failure.
func parseContentRange(header string) int64 {
	idx := strings.LastIndex(header, "/")
	if idx == -1 {
		return -1
	}
	totalStr := strings.TrimSpace(header[idx+1:])
	if totalStr == "*" {
		return -1
	}
	total, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil {
		return -1
	}
	return total
}
