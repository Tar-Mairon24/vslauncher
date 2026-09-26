package mods

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/andreyvit/jsonfix"
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