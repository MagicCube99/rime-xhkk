//go:build windows

package rime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"unsafe"
)

// DetectRimeDir detects the Rime user configuration directory on Windows.
// Order of priority:
//  1. RIME_DIR environment variable override (if set and exists).
//  2. Current working directory or parent directories if they contain flypy.dict.yaml or flypy_dicts (repo workspace priority).
//  3. Windows registry HKCU\Software\Rime\Weasel\RimeUserDir if user configured a custom path.
//  4. Default fallback to %APPDATA%\Rime.
func DetectRimeDir() string {
	// 1. Explicit environment variable override (useful for testing and CLI flags)
	if env := os.Getenv("RIME_DIR"); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return filepath.Clean(env)
		}
	}

	// 2. Checks Windows registry HKCU\Software\Rime\Weasel\RimeUserDir (highest system priority for active Weasel installation)
	if regUserDir := queryRegistryString(registry.CURRENT_USER, `Software\Rime\Weasel`, "RimeUserDir"); regUserDir != "" {
		regUserDir = os.ExpandEnv(regUserDir)
		if valid, _ := ValidateRimeDir(regUserDir); valid {
			return filepath.Clean(regUserDir)
		}
	}

	// 3. Checks standard %APPDATA%\Rime if it exists and is valid
	if appData := os.Getenv("APPDATA"); appData != "" {
		p := filepath.Join(appData, "Rime")
		if valid, _ := ValidateRimeDir(p); valid {
			return filepath.Clean(p)
		}
	}

	// 4. Fallback to current working directory or ancestor directories containing flypy.dict.yaml / flypy_dicts
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

	// 5. Default fallback to %APPDATA%\Rime
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "Rime")
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return filepath.Join(userProfile, "AppData", "Roaming", "Rime")
	}
	return `C:\Rime`
}

// DetectWeaselDeployer checks the Windows registry and standard Program Files paths
// to locate WeaselDeployer.exe.
// Returns the absolute executable path and true if found, or ("", false) if not found.
func DetectWeaselDeployer() (string, bool) {
	// 1. Checks Windows registry HKCU\Software\Rime\Weasel\WeaselRoot and HKLM\Software\Rime\Weasel\WeaselRoot
	regRoots := []struct {
		rootKey registry.Key
		subKey  string
	}{
		{registry.CURRENT_USER, `Software\Rime\Weasel`},
		{registry.LOCAL_MACHINE, `Software\Rime\Weasel`},
	}

	for _, rr := range regRoots {
		val := queryRegistryString(rr.rootKey, rr.subKey, "WeaselRoot")
		if val != "" {
			val = os.ExpandEnv(val)
			if deployer, ok := resolveDeployerExe(val); ok {
				return deployer, true
			}
		}
	}

	// 2. Checks common Program Files paths
	var baseDirs []string
	addBaseDir := func(d string) {
		d = strings.TrimSpace(d)
		if d == "" {
			return
		}
		for _, existing := range baseDirs {
			if strings.EqualFold(existing, d) {
				return
			}
		}
		baseDirs = append(baseDirs, d)
	}

	addBaseDir(os.Getenv("ProgramFiles"))
	addBaseDir(os.Getenv("ProgramFiles(x86)"))
	addBaseDir(os.Getenv("ProgramW6432"))
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		addBaseDir(filepath.Join(localApp, "Programs"))
	}
	addBaseDir(`C:\Program Files`)
	addBaseDir(`C:\Program Files (x86)`)

	for _, baseDir := range baseDirs {
		// Pattern A: <baseDir>\Rime\weasel-*\WeaselDeployer.exe
		patternA := filepath.Join(baseDir, "Rime", "weasel-*", "WeaselDeployer.exe")
		if matches, err := filepath.Glob(patternA); err == nil && len(matches) > 0 {
			sort.Slice(matches, func(i, j int) bool {
				return matches[i] > matches[j] // Highest version first
			})
			for _, m := range matches {
				if fileExists(m) {
					return m, true
				}
			}
		}

		// Pattern B: <baseDir>\Rime\*\WeaselDeployer.exe
		patternB := filepath.Join(baseDir, "Rime", "*", "WeaselDeployer.exe")
		if matches, err := filepath.Glob(patternB); err == nil && len(matches) > 0 {
			sort.Slice(matches, func(i, j int) bool {
				return matches[i] > matches[j]
			})
			for _, m := range matches {
				if fileExists(m) {
					return m, true
				}
			}
		}

		// Direct: <baseDir>\Rime\WeaselDeployer.exe
		direct := filepath.Join(baseDir, "Rime", "WeaselDeployer.exe")
		if fileExists(direct) {
			return direct, true
		}
	}

	// 3. Fallback: check PATH
	if p, err := exec.LookPath("WeaselDeployer.exe"); err == nil {
		if abs, errAbs := filepath.Abs(p); errAbs == nil && fileExists(abs) {
			return abs, true
		}
	}

	return "", false
}

// TriggerDeploy executes WeaselDeployer.exe with "/deploy" quietly (hiding window)
// with a 15-second timeout context so it doesn't hang.
// If deployerPath is empty, it attempts to auto-detect using DetectWeaselDeployer.
func TriggerDeploy(deployerPath string) error {
	deployerPath = strings.TrimSpace(deployerPath)
	if deployerPath == "" {
		detected, ok := DetectWeaselDeployer()
		if !ok {
			return errors.New("weasel deployer not found")
		}
		deployerPath = detected
	}

	if !fileExists(deployerPath) {
		return fmt.Errorf("deployer executable not found at %s", deployerPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, deployerPath, "/deploy")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("deploy timed out after 15s: %w", ctx.Err())
		}
		return fmt.Errorf("deploy execution failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func queryRegistryString(root registry.Key, path string, valueName string) string {
	accessModes := []uint32{
		registry.QUERY_VALUE,
		registry.QUERY_VALUE | registry.WOW64_32KEY,
		registry.QUERY_VALUE | registry.WOW64_64KEY,
	}

	for _, mode := range accessModes {
		k, err := registry.OpenKey(root, path, mode)
		if err == nil {
			val, _, err := k.GetStringValue(valueName)
			_ = k.Close()
			if err == nil && strings.TrimSpace(val) != "" {
				return strings.TrimSpace(val)
			}
		}
	}
	return ""
}

func resolveDeployerExe(root string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", false
	}

	if strings.EqualFold(filepath.Base(root), "WeaselDeployer.exe") && fileExists(root) {
		return root, true
	}

	candidate := filepath.Join(root, "WeaselDeployer.exe")
	if fileExists(candidate) {
		return candidate, true
	}

	return "", false
}

// IsProcessRunning checks whether an executable with the given process name is running on Windows.
func IsProcessRunning(processName string) bool {
	hSnapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(hSnapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(hSnapshot, &entry); err != nil {
		return false
	}

	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, processName) {
			return true
		}
		if err := windows.Process32Next(hSnapshot, &entry); err != nil {
			break
		}
	}
	return false
}

// DetectWeaselServer locates WeaselServer.exe and checks whether its service process is actively running.
// Returns (path, exists, running).
func DetectWeaselServer() (string, bool, bool) {
	var serverPath string
	deployer, ok := DetectWeaselDeployer()
	if ok {
		candidate := filepath.Join(filepath.Dir(deployer), "WeaselServer.exe")
		if fileExists(candidate) {
			serverPath = candidate
		}
	}

	if serverPath == "" {
		regRoots := []struct {
			rootKey registry.Key
			subKey  string
		}{
			{registry.CURRENT_USER, `Software\Rime\Weasel`},
			{registry.LOCAL_MACHINE, `Software\Rime\Weasel`},
		}
		for _, rr := range regRoots {
			val := queryRegistryString(rr.rootKey, rr.subKey, "WeaselRoot")
			if val != "" {
				candidate := filepath.Join(os.ExpandEnv(val), "WeaselServer.exe")
				if fileExists(candidate) {
					serverPath = candidate
					break
				}
			}
		}
	}

	exists := (serverPath != "")
	running := IsProcessRunning("WeaselServer.exe")
	return serverPath, exists, running
}
