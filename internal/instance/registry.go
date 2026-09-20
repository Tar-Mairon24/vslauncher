package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func registryPath() (string, error) {
	root, err := defaultInstancesRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "registry.json"), nil
}

func addToRegistry(name, path string, version string) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}

	entries = append(entries, registryEntry{
		Name:    name,
		Path:    path,
		Version: version,
	})

	registryPath, err := registryPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("writing registry file: %w", err)
	}

	return nil
}

func loadRegistry() ([]registryEntry, error) {
	registryPath, err := registryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(registryPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var entries []registryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}