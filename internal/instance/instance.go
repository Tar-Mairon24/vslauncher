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

func Create(ctx context.Context, name string, release versions.Release) (*Instance, error) {
	instancesRoot, err := defaultInstancesRoot()
	if err != nil {
		return nil, err
	}
	
	instanceDir := filepath.Join(instancesRoot, name)

	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		return nil, fmt.Errorf("instance %q already exists at %s", name, instanceDir)
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
