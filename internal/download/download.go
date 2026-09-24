package download

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tar-Mairon24/vslauncher/internal/versions"
)

type ProgressFactory func(size int64) io.Writer

type StepReporter interface {
	Step(label string)
}

type ExtractProgressFactory func(totalFiles int) StepReporter

type ProgressReporters struct {
	Download ProgressFactory
	Extract  ExtractProgressFactory
}

func FetchAndExtract(ctx context.Context, release versions.Release, destDir string, progress ProgressReporters) error {
	tmpfile, err := os.CreateTemp("", "vsl-download-*.tar.gz")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpfilePath := tmpfile.Name()
	defer os.Remove(tmpfilePath)

	if err := download(ctx, release.DownloadURL, tmpfile, progress.Download); err != nil {
		tmpfile.Close()
		return fmt.Errorf("downloading %s: %w", release.DownloadURL, err)
	}
	tmpfile.Close()

	if err := verifyChecksum(tmpfilePath, release.Checksum, release.ChecksumAlgo); err != nil {
		return fmt.Errorf("verifying checksum: %w", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}
	return extractTarGz(tmpfilePath, destDir, progress.Extract)
}

func download(ctx context.Context, url string, out *os.File, newProgress ProgressFactory) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	dest := io.Writer(out)
	if newProgress != nil {
		bar := newProgress(resp.ContentLength)
		if closer, ok := bar.(io.Closer); ok {
			defer closer.Close()
		}
		dest = io.MultiWriter(out, bar)
	}
	_, err = io.Copy(dest, resp.Body)
	return err
}

func verifyChecksum(path, expected, algo string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var hash hash.Hash
	switch strings.ToLower(algo) {
	case "md5":
		hash = md5.New()
	case "sha1":
		hash = sha1.New()
	case "sha256":
		hash = sha256.New()
	default:
		return fmt.Errorf("unsupported checksum algorithm: %q", algo)
	}

	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != expected {
		return fmt.Errorf("checksum mismatch (%s): got %s, want %s", algo, got, expected)
	}
	return nil
}

func extractTarGz(archivePath, destDir string, newExtractProgress ExtractProgressFactory) error {
	total, err := countTarEntries(archivePath)
	if err != nil {
		return fmt.Errorf("counting archive entries: %w", err)
	}

	var reporter StepReporter
	if newExtractProgress != nil {
		reporter = newExtractProgress(total)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			return fmt.Errorf("unsupported file type in archive: %v", header.Typeflag)
		}

		if reporter != nil {
			reporter.Step(header.Name)
		}
	}
}

func countTarEntries(archivePath string) (int, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return 0, err
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)
	count := 0
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		count++
	}
}
