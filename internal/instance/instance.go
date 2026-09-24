package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Tar-Mairon24/vslauncher/internal/download"
	"github.com/Tar-Mairon24/vslauncher/internal/versions"
)

func Create(ctx context.Context, name string, release versions.Release, customPath string, createDesktopFile bool, progress download.ProgressReporters) (*Instance, error) {	var instancesRoot string
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

	if _, err := FindByName(name); err == nil {
		return nil, fmt.Errorf("instance %q already exists", name)
	} else if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("checking for existing instance: %w", err)
	}


	if err := download.FetchAndExtract(ctx, release, instanceDir, progress); err != nil {
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
		IconPath:    resolveIconPath(instanceDir),
		InstalledAt: time.Now(),
	}

	if createDesktopFile {
		desktopPath, err := CreateDesktopFile(inst)
		if err != nil {
			return nil, fmt.Errorf("creating desktop file: %w", err)
		}
		inst.DesktopPath = desktopPath
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
	instToRemove, err := FindByName(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(instToRemove.Path); err != nil {
		return fmt.Errorf("removing instance directory %s: %w", instToRemove.Path, err)
	}

	if err := removeFromRegistry(name); err != nil {
		fmt.Fprintf(os.Stderr, "warning: instance files removed, but could not clean up registry entry: %v\n", err)
	}

	if err := removeDesktopFile(instToRemove); err != nil {
		fmt.Fprintf(os.Stderr, "warning: instance files removed, but could not remove desktop file: %v\n", err)
	}

	return nil
}

func Update(name string, newRelease versions.Release, progress download.ProgressReporters) error {
	instToUpdate, err := FindByName(name)
	if err != nil {
		return err
	}

	gameDir := filepath.Join(instToUpdate.Path, "vintagestory")
	oldDir := filepath.Join(instToUpdate.Path, ".vintagestory.old")

	if err := os.RemoveAll(oldDir); err != nil {
		return fmt.Errorf("removing stale backup directory: %w", err)
	}

	tmpParent, err := os.MkdirTemp(instToUpdate.Path, "vslauncher-update-*")
	if err != nil {
		return fmt.Errorf("creating temporary directory for update: %w", err)
	}
	defer os.RemoveAll(tmpParent)

	if err := download.FetchAndExtract(context.Background(), newRelease, tmpParent, progress); err != nil {
		return fmt.Errorf("downloading and extracting new release: %w", err)
	}

	extracted := filepath.Join(tmpParent, "vintagestory")
	if _, err := os.Stat(extracted); os.IsNotExist(err) {
		return fmt.Errorf("extracted release does not contain expected 'vintagestory' directory")
	} else if err != nil {
		return fmt.Errorf("checking extracted release: %w", err)
	}

	if err := os.Rename(gameDir, oldDir); err != nil {
		return fmt.Errorf("staging old game files: %w", err)
	}
	if err := os.Rename(extracted, gameDir); err != nil {
		os.Rename(oldDir, gameDir)
		return fmt.Errorf("installing new game files: %w", err)
	}

	if err := os.RemoveAll(oldDir); err != nil {
		return fmt.Errorf("removing old game files: %w", err)
	}

	instToUpdate.Version = newRelease.Version
	instToUpdate.Channel = newRelease.Channel
	now := time.Now()
	instToUpdate.UpdatedAt = &now
	return save(instToUpdate)
}

func CreateDesktopFile(inst *Instance) (string, error) {
	if inst.Platform != "linux" {
		return "", fmt.Errorf("desktop file creation is only supported on Linux")
	}

	linuxDesktopPath, err := desktopFileDir()
	if err != nil {
		return "", fmt.Errorf("getting desktop file directory: %w", err)
	}
	if err := os.MkdirAll(linuxDesktopPath, 0755); err != nil {
		return "", fmt.Errorf("creating desktop file directory: %w", err)
	}
	desktopPath := filepath.Join(linuxDesktopPath, fmt.Sprintf("vslauncher-%s.desktop", inst.Name))
	
	file, err := os.Create(desktopPath)
	if err != nil {
		return "", fmt.Errorf("creating desktop file: %w", err)
	}
	defer file.Close()

	inst.DesktopPath = desktopPath
	
	data := desktopData{Instance: inst, BinaryPath: filepath.Join(inst.Path, "vintagestory", "Vintagestory")}
	if err := desktopLinuxTemplate.Execute(file, data); err != nil {
		return "", fmt.Errorf("writing desktop file: %w", err)
	}

	return desktopPath, nil
}

func resolveIconPath(instanceDir string) string {
	assetsDir := filepath.Join(instanceDir, "vintagestory", "assets")

	pngPath := filepath.Join(assetsDir, "gameicon.png")
	if _, err := os.Stat(pngPath); err == nil {
		return pngPath
	}

	xpmPath := filepath.Join(assetsDir, "gameicon.xpm")
	if _, err := os.Stat(xpmPath); err == nil {
		convertedPath := filepath.Join(assetsDir, "gameicon-converted.png")
		if err := convertXPMtoPNG(xpmPath, convertedPath); err == nil {
			return convertedPath
		}
		return xpmPath
	}

	return ""
}

func removeDesktopFile(inst *Instance) error {
	if inst.DesktopPath == "" {
		return nil
	}
	if err := os.Remove(inst.DesktopPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing desktop file: %w", err)
	}
	return nil
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

func FindByName(name string) (*Instance, error) {
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
		return nil, fmt.Errorf("instance %q: %w", name, ErrNotFound)
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

func desktopFileDir() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "applications"), nil
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

func convertXPMtoPNG(xpmPath, pngPath string) error {
	converter := "convert"
	if _, err := exec.LookPath("magick"); err == nil {
		converter = "magick"
	} else if _, err := exec.LookPath("convert"); err != nil {
		return fmt.Errorf("no image converter (convert/magick) found on PATH, install ImageMagick to convert XPM icons to PNG: %w", err)
	}
	cmd := exec.Command(converter, xpmPath, pngPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("converting xpm to png: %w", err)
	}
	return nil
}