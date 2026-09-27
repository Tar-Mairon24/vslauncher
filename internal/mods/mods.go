package mods

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andreyvit/jsonfix"

	"github.com/Tar-Mairon24/vslauncher/internal/backup"
	"github.com/Tar-Mairon24/vslauncher/internal/download"
)

func ScanInstalled(dataPath string) ([]InstalledMod, error) {
	modsDir := filepath.Join(dataPath, "Mods")

	entries, err := os.ReadDir(modsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading mods directory %s: %w", modsDir, err)
	}

	var mods []InstalledMod
	for _, entry := range entries {
		path := filepath.Join(modsDir, entry.Name())

		var info *ModInfo
		var readErr error

		switch {
		case entry.IsDir():
			info, readErr = readModInfoFromDir(path)
		case strings.EqualFold(filepath.Ext(entry.Name()), ".zip"):
			info, readErr = readModInfoFromZip(path)
		default:
			continue
		}

		if readErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read mod info from %s: %v\n", entry.Name(), readErr)
			continue
		}
		mods = append(mods, InstalledMod{Info: *info, Path: path})
	}

	disabled, err := loadDisabledMods(dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read mod enable/disable state: %v\n", err)
		disabled = map[string]bool{}
	}

	for i := range mods {
		mods[i].Enabled = !disabled[mods[i].Info.QueryID()]
	}
	return mods, nil
}

func UpdateMods(ctx context.Context, opts UpdateModParams) ([]UpdateResult, error) {
	checked, err := CheckForUpdates(ctx, opts.Installed, opts.GameVersion, opts.TargetModIDs, opts.ExcludedModIDs)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}

	upgradeIDs := upgradeableModIDs(checked)
	if len(upgradeIDs) == 0 {
		return nil, nil
	}

	if opts.BackupDir != "" {
		if err := backupBeforeUpdate(opts.BackupDir, opts.InstName, opts.ModsDir, opts.MaxBackups); err != nil {
			return nil, fmt.Errorf("creating backup: %w", err)
		}
	}

	upgrades, err := FetchInstallInfo(ctx, upgradeIDs, opts.GameVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving upgrade downloads: %w", err)
	}

	results := downloadAndReplaceMods(ctx, upgrades, opts.Installed, opts.ModsDir, opts.NewProgress)

	clearCache(opts.DataPath)
	return results, nil
}

func CheckForUpdates(ctx context.Context, installed []InstalledMod, gameVersion string, targetModIDs []string, excludedModIDs []string) ([]InstallInfoResult, error) {
	excluded := toSet(excludedModIDs)
	targets := toSet(targetModIDs)

	var checkIDs []string
	for _, m := range installed {
		if excluded[m.Info.ModID] {
			continue
		}
		if len(targets) > 0 && !targets[m.Info.ModID] {
			continue
		}
		checkIDs = append(checkIDs, m.Info.QueryID())
	}
	if len(checkIDs) == 0 {
		return nil, nil
	}

	results, err := FetchInstallInfo(ctx, checkIDs, gameVersion)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	return results, nil
}

func downloadAndReplaceMods(ctx context.Context, upgrades []InstallInfoResult, installed []InstalledMod, modsDir string, newProgress download.MultiProgressFactory) []UpdateResult {
	byModID := make(map[string]InstalledMod, len(installed))
	for _, m := range installed {
		byModID[m.Info.ModID] = m
	}

	results := make([]UpdateResult, 0, len(upgrades))
	for _, r := range upgrades {
		old := byModID[r.Name]
		res := UpdateResult{ModID: r.Name, OldVersion: old.Info.Version}

		if !r.Available() {
			res.Error = fmt.Sprintf("error %d: %s", r.ErrorCode, r.RetractionReason)
			results = append(results, res)
			continue
		}

		var progress download.ProgressFactory
		if newProgress != nil {
			progress = newProgress(r.Name)
		}

		newPath, err := downloadMod(ctx, r, modsDir, progress)
		if err != nil {
			res.Error = fmt.Sprintf("downloading: %v", err)
			results = append(results, res)
			continue
		}

		if filepath.Ext(old.Path) == ".zip" && old.Path != newPath {
			if err := os.Remove(old.Path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: downloaded %s but could not remove old file %s: %v\n", newPath, old.Path, err)
			}
		}

		if info, err := readModInfoFromZip(newPath); err != nil {
			res.NewVersion = extractVersionFromFilename(newPath)
		} else {
			res.NewVersion = info.Version
		}
		results = append(results, res)
	}
	return results
}

func downloadMod(ctx context.Context, result InstallInfoResult, modsDir string, newProgress download.ProgressFactory) (string, error) {
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

func upgradeableModIDs(checked []InstallInfoResult) []string {
	var ids []string
	for _, r := range checked {
		if r.Available() && r.RecommendedUpgrade != "" {
			ids = append(ids, r.Name)
		}
	}
	return ids
}

func backupBeforeUpdate(backupDir string, instName string, modsDir string, maxBackups int) error {
	label := fmt.Sprintf("%s_mods", instName)
	if _, err := backup.Create(modsDir, backupDir, label); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}
	if err := backup.Prune(backupDir, label, maxBackups); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not prune old backups: %v\n", err)
	}
	return nil
}


func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, i := range items {
		s[i] = true
	}
	return s
}

func clearCache(dataPath string) error {
	return os.RemoveAll(filepath.Join(dataPath, "Cache"))
}

func extractVersionFromFilename(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func readModInfoFromDir(dir string) (*ModInfo, error) {
	data, err := os.ReadFile(filepath.Join(dir, "modinfo.json"))
	if err != nil {
		return nil, err
	}
	return parseModInfo(data)
}

func readModInfoFromZip(zipPath string) (*ModInfo, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.EqualFold(f.Name, "modinfo.json") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening modinfo.json in zip: %w", err)
			}
			defer rc.Close()

			var buf []byte
			buf, err = io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("reading modinfo.json: %w", err)
			}
			return parseModInfo(buf)
		}
	}
	return nil, fmt.Errorf("modinfo.json not found in zip")
}

func parseModInfo(data []byte) (*ModInfo, error) {
	cleaned := jsonfix.Bytes(data)
	var info ModInfo
	if err := json.Unmarshal(cleaned, &info); err != nil {
		return nil, fmt.Errorf("parsing modinfo.json: %w", err)
	}
	info.Side = strings.ToUpper(info.Side)
	return &info, nil
}

func loadDisabledMods(dataPath string) (map[string]bool, error) {
	path := filepath.Join(dataPath, "clientsettings.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading clientsettings.json: %w", err)
	}

	var settings clientSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing clientsettings.json: %w", err)
	}

	disabled := make(map[string]bool, len(settings.StringListSettings.DisabledMods))
	for _, entry := range settings.StringListSettings.DisabledMods {
		disabled[entry] = true
	}
	return disabled, nil
}
