package downloader

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ChunkState represents the current state of a download chunk.
type ChunkState int

const (
	ChunkPending     ChunkState = iota // Waiting to start
	ChunkDownloading                   // Actively downloading
	ChunkComplete                      // Finished successfully
	ChunkFailed                        // Failed permanently
	ChunkRetrying                      // Retrying after a transient failure
)

// ChunkStatus holds the runtime status of a single download chunk.
type ChunkStatus struct {
	ID         int
	Start      int64
	End        int64 // inclusive byte offset
	Downloaded int64
	State      ChunkState
	Retries    int
}

// Progress represents a point-in-time snapshot of download progress.
type Progress struct {
	TotalBytes      int64
	DownloadedBytes int64
	Speed           float64       // current speed (bytes/sec), EMA-smoothed
	AvgSpeed        float64       // average speed over the entire download
	ETA             time.Duration // estimated time remaining
	Percentage      float64       // 0-100
	Elapsed         time.Duration
	Chunks          []ChunkStatus
	ActiveChunks    int
	IsComplete      bool
}

// ProgressCallback is invoked periodically with download progress updates.
// Implementations must be safe for concurrent calls.
type ProgressCallback func(Progress)

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

// FormatSpeed returns a human-readable representation of bytes/sec.
func FormatSpeed(bps float64) string {
	switch {
	case bps >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB/s", bps/(1024*1024*1024))
	case bps >= 1024*1024:
		return fmt.Sprintf("%.1f MB/s", bps/(1024*1024))
	case bps >= 1024:
		return fmt.Sprintf("%.1f KB/s", bps/1024)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

// FormatBytes returns a human-readable representation of a byte count.
func FormatBytes(b int64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.2f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatETA returns a human-readable ETA string.
func FormatETA(d time.Duration) string {
	if d <= 0 {
		return "calculating..."
	}
	if d > 24*time.Hour {
		return ">24h"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// ---------------------------------------------------------------------------
// progressTracker – internal, thread-safe progress aggregator
// ---------------------------------------------------------------------------

type progressTracker struct {
	totalBytes int64
	downloaded atomic.Int64
	startTime  time.Time

	mu     sync.RWMutex
	chunks []ChunkStatus

	// EMA-based speed estimation
	speedMu        sync.Mutex
	lastSpeedAt    time.Time
	lastSpeedBytes int64
	emaSpeed       float64

	callback     ProgressCallback
	callbackRate time.Duration
	lastCallback time.Time
}

func newProgressTracker(totalBytes int64, numChunks int, cb ProgressCallback) *progressTracker {
	now := time.Now()
	return &progressTracker{
		totalBytes:   totalBytes,
		startTime:    now,
		chunks:       make([]ChunkStatus, numChunks),
		lastSpeedAt:  now,
		callback:     cb,
		callbackRate: 80 * time.Millisecond,
		lastCallback: now,
	}
}

// addBytes records newly downloaded bytes (called from any goroutine).
func (pt *progressTracker) addBytes(n int64) {
	pt.downloaded.Add(n)
	pt.maybeNotify()
}

// initChunk sets the byte range for a chunk before it starts downloading.
func (pt *progressTracker) initChunk(id int, start, end int64) {
	pt.mu.Lock()
	if id < len(pt.chunks) {
		pt.chunks[id] = ChunkStatus{ID: id, Start: start, End: end, State: ChunkPending}
	}
	pt.mu.Unlock()
}

func (pt *progressTracker) setChunkState(id int, state ChunkState) {
	pt.mu.Lock()
	if id < len(pt.chunks) {
		pt.chunks[id].State = state
	}
	pt.mu.Unlock()
}

func (pt *progressTracker) setChunkDownloaded(id int, bytes int64) {
	pt.mu.Lock()
	if id < len(pt.chunks) {
		pt.chunks[id].Downloaded = bytes
	}
	pt.mu.Unlock()
}

func (pt *progressTracker) incChunkRetries(id int) {
	pt.mu.Lock()
	if id < len(pt.chunks) {
		pt.chunks[id].Retries++
	}
	pt.mu.Unlock()
}

// currentSpeed returns the EMA-smoothed speed in bytes/sec.
func (pt *progressTracker) currentSpeed() float64 {
	pt.speedMu.Lock()
	defer pt.speedMu.Unlock()

	now := time.Now()
	elapsed := now.Sub(pt.lastSpeedAt).Seconds()
	if elapsed < 0.1 {
		return pt.emaSpeed
	}

	currentBytes := pt.downloaded.Load()
	delta := float64(currentBytes - pt.lastSpeedBytes)
	instantSpeed := delta / elapsed

	const alpha = 0.3 // smoothing factor
	if pt.emaSpeed == 0 {
		pt.emaSpeed = instantSpeed
	} else {
		pt.emaSpeed = alpha*instantSpeed + (1-alpha)*pt.emaSpeed
	}

	pt.lastSpeedAt = now
	pt.lastSpeedBytes = currentBytes
	return pt.emaSpeed
}

// snapshot produces a consistent Progress value.
func (pt *progressTracker) snapshot() Progress {
	downloaded := pt.downloaded.Load()
	elapsed := time.Since(pt.startTime)
	speed := pt.currentSpeed()

	var avgSpeed float64
	if elapsed.Seconds() > 0 {
		avgSpeed = float64(downloaded) / elapsed.Seconds()
	}

	var eta time.Duration
	remaining := pt.totalBytes - downloaded
	if speed > 0 && remaining > 0 {
		eta = time.Duration(float64(remaining)/speed) * time.Second
	}

	var pct float64
	if pt.totalBytes > 0 {
		pct = float64(downloaded) / float64(pt.totalBytes) * 100
	}
	if pct > 100 {
		pct = 100
	}

	pt.mu.RLock()
	chunks := make([]ChunkStatus, len(pt.chunks))
	copy(chunks, pt.chunks)
	active := 0
	for _, c := range chunks {
		if c.State == ChunkDownloading {
			active++
		}
	}
	pt.mu.RUnlock()

	return Progress{
		TotalBytes:      pt.totalBytes,
		DownloadedBytes: downloaded,
		Speed:           speed,
		AvgSpeed:        avgSpeed,
		ETA:             eta,
		Percentage:      pct,
		Elapsed:         elapsed,
		Chunks:          chunks,
		ActiveChunks:    active,
		IsComplete:      pt.totalBytes > 0 && downloaded >= pt.totalBytes,
	}
}

// maybeNotify fires the callback if enough time has elapsed since the last one.
func (pt *progressTracker) maybeNotify() {
	if pt.callback == nil {
		return
	}
	now := time.Now()
	if now.Sub(pt.lastCallback) >= pt.callbackRate {
		pt.lastCallback = now
		pt.callback(pt.snapshot())
	}
}

// notifyComplete fires a final callback with IsComplete=true.
func (pt *progressTracker) notifyComplete() {
	if pt.callback == nil {
		return
	}
	snap := pt.snapshot()
	snap.IsComplete = true
	pt.callback(snap)
}
