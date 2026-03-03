package downloader

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

// Downloader manages file downloads from Civitai.
type Downloader struct {
	http *resty.Client
	cfg  *config.Config
}

func NewDownloader(cfg *config.Config) *Downloader {
	r := resty.New()
	// Disable automatic redirects to intercept the filename from headers if needed,
	// or let it redirect but capture the response. Let's rely on standard redirect following.
	return &Downloader{
		http: r,
		cfg:  cfg,
	}
}

// DownloadToWriter downloads a file and reports progress to the provided io.Writer.
// The caller is responsible for providing a writer that can interpret the progress streams.
func (d *Downloader) DownloadToWriter(ctx context.Context, versionID int, outputDir string, progressWriter io.Writer) (string, error) {
	downloadUrl := fmt.Sprintf("https://civitai.com/api/download/models/%d", versionID)

	// Prepare the request
	req := d.http.R().
		SetContext(ctx).
		SetDoNotParseResponse(true) // We want the raw reader

	if d.cfg.APIKey != "" {
		req.SetQueryParam("token", d.cfg.APIKey)
	}

	log.Debug().Msgf("Downloading from %s", downloadUrl)
	resp, err := req.Get(downloadUrl)
	if err != nil {
		return "", fmt.Errorf("failed to initiate download: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("failed to download (status %d): %s", resp.StatusCode(), resp.String())
	}

	// Determine filename from Content-Disposition header
	filename := extractFilename(resp.Header().Get("Content-Disposition"))
	if filename == "" {
		filename = fmt.Sprintf("model_%d.safetensors", versionID) // Fallback
	}

	// Create output dir if needed
	if outputDir == "" {
		outputDir = d.cfg.DefaultDownloadDir
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	outPath := filepath.Join(outputDir, filename)

	// Setup file output
	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Parse content length for progress bar if available
	contentLength := resp.RawResponse.ContentLength
	if progressWriter != nil {
		// If it implements ChangeMax64, we can set the size
		if maxSetter, ok := progressWriter.(interface{ ChangeMax64(int64) }); ok {
			maxSetter.ChangeMax64(contentLength)
		}

		_, err = io.Copy(io.MultiWriter(out, progressWriter), resp.RawBody())
	} else {
		_, err = io.Copy(out, resp.RawBody())
	}

	if err != nil {
		return "", fmt.Errorf("error during download writing: %w", err)
	}

	return outPath, nil
}

// extractFilename attempts to parse the filename from the Content-Disposition header
func extractFilename(cd string) string {
	if cd == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}
	filename := params["filename"]
	// Some headers might use filename* but we'll try the simple path and sanitize
	return strings.Trim(filename, "\"")
}
