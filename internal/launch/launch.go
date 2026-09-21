package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
)

func LaunchInstance(ctx context.Context, inst *instance.Instance, options Options) error {
	if inst.DataPath == "" {
		return fmt.Errorf("instance %q has no data path set", inst.Name)
	}
	if err := os.MkdirAll(inst.DataPath, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	dir := filepath.Join(inst.Path, "vintagestory")

	binary := filepath.Join(dir, "Vintagestory")

	info, err := os.Stat(binary)
	if err != nil {
		return fmt.Errorf("checking instance binary: %w", err)
	}
	if info.Mode()&0111 == 0 {
		if err := os.Chmod(binary, 0755); err != nil {
			return fmt.Errorf("making instance binary executable: %w", err)
		}
	}

	args := buildArgs(inst, options)

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin


	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running instance %q: %w", inst.Name, err)
	}
	return nil
}

func buildArgs(inst *instance.Instance, options Options) []string {
	args := []string{"--dataPath", inst.DataPath}

	if options.World != "" {
		args = append(args, "--openWorld", options.World)
	}
	
	if options.Connect != "" {
		args = append(args, "--connect", options.Connect)
	}

	if options.Password != "" {
		args = append(args, "--password", options.Password)
	}

	if options.RandomWorld {
		args = append(args, "--rndmWorld")
	}

	if options.PlayStyle != "" {
		args = append(args, "--playStyle", options.PlayStyle)
	}

	if options.InstallMod != "" {
		args = append(args, "--installMod", options.InstallMod)
	}
	
	return args
}