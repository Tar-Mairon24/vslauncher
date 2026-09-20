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

func FetchAndExtract(ctx context.Context, release versions.Release, destDir string) error {
	tmpfile, err := os.CreateTemp("", "vsl-download-*.tar.gz")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpfilePath := tmpfile.Name()
	defer os.Remove(tmpfilePath)

	if err := download(ctx, release.DownloadURL, tmpfile); err != nil {
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

	if err := extractTarGz(tmpfilePath, destDir); err != nil {
		return fmt.Errorf("extracting archive %s: %w", tmpfilePath, err)
	}

	return nil
}

func download(ctx context.Context, url string, out *os.File) error {
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

	_, err = io.Copy(out, resp.Body)
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

func extractTarGz(archivePath, destDir string) error {
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
	}
}

