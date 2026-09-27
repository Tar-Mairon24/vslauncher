package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Create(srcDir string, backupDir string, label string) (string, error) {
	if _, err := os.Stat(srcDir); err != nil {
		return "", fmt.Errorf("nothing to backup, %s does not exist", srcDir)
	}

	if err := os.Mkdir(backupDir, 0755); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	archiveName := fmt.Sprintf("%s_%s.zip", label, timestamp)
	archivePath := filepath.Join(backupDir, archiveName)

	out, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating backup archive: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		zf, err := zw.Create(filepath.ToSlash(relPath))
		if err != nil {
			return fmt.Errorf("creating zip entry for %s: %w", relPath, err)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", path, err)
		}
		defer file.Close()

		_, err = io.Copy(zf, file)
		if err != nil {
			os.Remove(archivePath)
			return fmt.Errorf("archiving %s to zip: %w", path, err)
		}

		return err
	})

	if err != nil {
		os.Remove(archivePath)
		return "", fmt.Errorf("creating backup: %w", err)
	}

	return archivePath, nil
}

func Prune(backupDir string, label string, maxBackups int) error {
	if maxBackups <= 0 {
		return nil
	}

	entries, err := os.ReadDir(backupDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading backup directory: %w", err)
	}

	prefix := label + "_"
	var matches []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".zip") {
			matches = append(matches, entry)
		}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].Name() > matches[j].Name() })

	for i := maxBackups; i < len(matches); i++ {
		path := filepath.Join(backupDir, matches[i].Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("warning: could not prune old backup %s: %v", path, err)
		}
	}
	return nil
}