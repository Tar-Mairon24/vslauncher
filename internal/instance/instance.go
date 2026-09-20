package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Tar-Mairon24/vslauncher/internal/download"
	"github.com/Tar-Mairon24/vslauncher/internal/versions"
)

func Create(ctx context.Context, name string, release versions.Release, customPath string) (*Instance, error) {
	var instancesRoot string
	var err error
	isCustomPath := customPath != ""

	if isCustomPath {
		instancesRoot = customPath
	} else {
		instancesRoot, err = defaultInstancesRoot()
		if err != nil {
			return nil, err
		}
	}

	instanceDir := filepath.Join(instancesRoot, name)

	if _, err := os.Stat(instanceDir); err == nil {
		return nil, fmt.Errorf("instance %q already exists at %s", name, instanceDir)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking instance path %s: %w", instanceDir, err)
	}

	if err := download.FetchAndExtract(ctx, release, instanceDir); err != nil {
		return nil, fmt.Errorf("installing instance %q: %w", name, err)
	}

	inst := &Instance{
		Name:        name,
		Version:     release.Version,
		Channel:     release.Channel,
		Platform:    release.Platform,
		Path:        instanceDir,
		InstalledAt: time.Now(),
	}

	if err := save(inst); err != nil {
		return nil, fmt.Errorf("saving instance %q metadata: %w", name, err)
	}

	if isCustomPath {
		if err := addToRegistry(name, instanceDir, release.Version); err != nil {
			return nil, fmt.Errorf("adding instance %q to registry: %w", name, err)
		}
		fmt.Fprintf(os.Stderr,
			"warning: %q is tracked via a separate registry file, not the default instances folder.\n"+
				"If the registry is lost or corrupted, this instance may not show up when listing instances.\n"+
				"even though its files remain on disk at %s.\n", name, instanceDir)
	}

	return inst, nil
}

func save(inst *Instance) error {
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return err
	}

	metaPath := filepath.Join(inst.Path, "instance.json")
	return os.WriteFile(metaPath, data, 0644)
}

func defaultInstancesRoot() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting user home directory: %w", err)
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataHome, "vslauncher", "instances"), nil
}
