package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Tar-Mairon24/vslauncher/internal/instance"
)

func LaunchInstance(ctx context.Context, inst *instance.Instance) error {
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

	args := []string{"--dataPath", inst.DataPath}

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