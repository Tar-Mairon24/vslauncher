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

	if err := os.MkdirAll(filepath.Join(instanceDir, "data"), 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	inst := &Instance{
		Name:        name,
		Version:     release.Version,
		Channel:     release.Channel,
		Platform:    release.Platform,
		Path:        instanceDir,
		DataPath:    filepath.Join(instanceDir, "data"),
		InstalledAt: time.Now(),
	}

	if err := save(inst); err != nil {
		return nil, fmt.Errorf("saving instance %q metadata: %w", name, err)
	}

	if isCustomPath {
		if err := addToRegistry(name, instanceDir); err != nil {
			return nil, fmt.Errorf("adding instance %q to registry: %w", name, err)
		}
		fmt.Fprintf(os.Stderr,
			"warning: %q is tracked via a separate registry file, not the default instances folder.\n"+
				"If the registry is lost or corrupted, this instance may not show up when listing instances.\n"+
				"even though its files remain on disk at %s.\n", name, instanceDir)
	}

	return inst, nil
}

func List() ([]Instance, error) {
	root, err := defaultInstancesRoot()
	if err != nil {
		return nil, err
	}

	instances, err := scanDir(root)
	if err != nil {
		return nil, err
	}

	knownPaths := make(map[string]bool, len(instances))
	for _, inst := range instances {
		knownPaths[inst.Path] = true
	}

	entries, err := loadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read registry, custom-path instances may be missing: %v\n", err)
		return instances, nil
	}

	for _, entry := range entries {
		if knownPaths[entry.Path] {
			continue
		}
		inst, err := load(entry.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load instance %q from registry path %s: %v\n", entry.Name, entry.Path, err)
			continue
		}
		instances = append(instances, *inst)
		knownPaths[entry.Path] = true
	}
	
	return instances, nil
}

func Remove(name string) error {
	instToRemove, err := findByName(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(instToRemove.Path); err != nil {
		return fmt.Errorf("removing instance directory %s: %w", instToRemove.Path, err)
	}

	if err := removeFromRegistry(name); err != nil {
		return fmt.Errorf("removing instance %q from registry: %w", name, err)
	}

	return nil
}

func Update(name string, newRelease versions.Release) error {
	instToUpdate, err := findByName(name)
	if err != nil {
		return err
	}

	gameDir := filepath.Join(instToUpdate.Path, "vintagestory")

	if err := os.RemoveAll(gameDir); err != nil {
		return fmt.Errorf("removing old game files: %w", err)
	}

	if err := download.FetchAndExtract(context.Background(), newRelease, instToUpdate.Path); err != nil {
		return fmt.Errorf("downloading and extracting new release: %w", err)
	}

	instToUpdate.Version = newRelease.Version
	instToUpdate.Channel = newRelease.Channel
	instToUpdate.DataPath = filepath.Join(instToUpdate.Path, "data")
	instToUpdate.UpdatedAt = time.Now()

	return save(instToUpdate)
}

func scanDir(root string) ([]Instance, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading instances root %s: %w", root, err)
	}

	var instances []Instance
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		instPath := filepath.Join(root, entry.Name())
		inst, err := load(instPath)
		if err != nil {
			continue
		}
		instances = append(instances, *inst)
	}

	return instances, nil
}

func load(path string) (*Instance, error) {
	data, err := os.ReadFile(filepath.Join(path, "instance.json"))
	if err != nil {
		return nil, fmt.Errorf("reading instance metadata: %w", err)
	}

	var inst Instance
	if err := json.Unmarshal(data, &inst); err != nil {
		return nil, fmt.Errorf("parsing instance metadata: %w", err)
	}

	return &inst, nil
}

func findByName(name string) (*Instance, error) {
	instances, err := List()
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}

	var instToFind *Instance
	for _, inst := range instances {
		if inst.Name == name {
			instToFind = &inst
			break
		}
	}

	if instToFind == nil {
		return nil, fmt.Errorf("instance %q not found", name)
	}
	return instToFind, nil
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
