package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
)

func DataHome() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting user home dir: %w", err)	
	}
	return filepath.Join(home, ".local", "share"), nil
}

func ConfigHome() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting user home dir: %w", err)
	}
	return filepath.Join(home, ".config"), nil
}

func DesktopFileDir() (string, error) {
	dataHome, err := DataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataHome, "applications"), nil
}

func DefaultInstancesRoot() (string, error) {
	dataHome, err := DataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataHome, "vslauncher", "instances"), nil
}