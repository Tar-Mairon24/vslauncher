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

func addToRegistry(name, path string) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}

	entries = append(entries, registryEntry{
		Name:    name,
		Path:    path,
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

func removeFromRegistry(name string) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}

	filtered := entries[:0]
	for _, entry := range entries {
		if entry.Name != name {
			filtered = append(filtered, entry)
		}
	}

	registryPath, err := registryPath()
	if err != nil {
		return err
	}

	if len(filtered) == 0 {
		if err := os.Remove(registryPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing registry file: %w", err)
		}
		return nil
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("writing registry file: %w", err)
	}

	return nil
}