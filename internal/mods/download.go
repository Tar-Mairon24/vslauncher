package mods

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Tar-Mairon24/vslauncher/internal/download"
)

func DownloadMod(ctx context.Context, result InstallInfoResult, modsDir string, newProgress download.ProgressFactory) (string, error) {
	if !result.Available() {
		return "", fmt.Errorf("unavailable (error %d): %s", result.ErrorCode, result.RetractionReason)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL(result.FileURL), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading mod: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return "", fmt.Errorf("creating mods directory: %w", err)
	}

	destPath := filepath.Join(modsDir, result.FileName)
	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("creating %s: %w", destPath, err)
	}
	defer out.Close()

	dest := io.Writer(out)
	if newProgress != nil {
		bar := newProgress(resp.ContentLength)
		if closer, ok := bar.(io.Closer); ok {
			defer closer.Close()
		}
		dest = io.MultiWriter(out, bar)
	}

	if _, err := io.Copy(dest, resp.Body); err != nil {
		return "", fmt.Errorf("writing to %s: %w", destPath, err)
	}

	return destPath, nil
}