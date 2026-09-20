package platform

import (
	"fmt"
	"runtime"
)

func Detect() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return "linux", nil
	case "windows":
		return "windows", nil
	case "darwin":
		switch runtime.GOARCH {
		case "arm64":
			return "mac-arm64", nil
		case "amd64":
			return "mac-x64", nil
		default:
			return "", fmt.Errorf("unsupported darwin arch: %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}