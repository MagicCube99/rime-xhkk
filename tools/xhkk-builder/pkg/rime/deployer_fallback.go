//go:build !windows

package rime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// DetectRimeDir returns the detected Rime user directory on non-Windows platforms (macOS / Linux).
// It checks:
//  1. RIME_DIR environment variable override (if set and exists).
//  2. Current working directory or parent directories if they contain flypy.dict.yaml or flypy_dicts.
//  3. Platform default directory (macOS: ~/Library/Rime; Linux: ~/.config/rime or ~/.local/share/rime).
func DetectRimeDir() string {
	if env := os.Getenv("RIME_DIR"); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return filepath.Clean(env)
		}
	}

	// Check current directory / parent paths (repo workspace priority)
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 8; i++ {
			if hasRimeDicts(dir) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Rime")
	}

	// Linux / BSD
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	linuxPath := filepath.Join(xdgConfig, "rime")
	if info, err := os.Stat(linuxPath); err == nil && info.IsDir() {
		return linuxPath
	}

	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		xdgData = filepath.Join(home, ".local", "share")
	}
	linuxDataPath := filepath.Join(xdgData, "rime")
	if info, err := os.Stat(linuxDataPath); err == nil && info.IsDir() {
		return linuxDataPath
	}

	return linuxPath
}

// DetectWeaselDeployer returns false on non-Windows platforms as Weasel is Windows-specific.
func DetectWeaselDeployer() (string, bool) {
	return "", false
}

// TriggerDeploy returns an error on non-Windows platforms.
func TriggerDeploy(deployerPath string) error {
	return errors.New("weasel deployer is only supported on Windows")
}

// DetectWeaselServer returns false on non-Windows platforms as Weasel is Windows-specific.
func DetectWeaselServer() (string, bool, bool) {
	return "", false, false
}
