package rime

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectRimeDir(t *testing.T) {
	dir := DetectRimeDir()
	if dir == "" {
		t.Log("No Rime dir detected (may be running in CI)")
	} else {
		t.Logf("Detected Rime dir: %s", dir)
	}

	// Test environment variable override
	tmpDir := t.TempDir()
	t.Setenv("RIME_DIR", tmpDir)
	overrideDir := DetectRimeDir()
	if overrideDir != filepath.Clean(tmpDir) {
		t.Fatalf("expected RIME_DIR override %s, got %s", tmpDir, overrideDir)
	}
}

func TestHasRimeDicts(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty dir should return false
	if hasRimeDicts(tmpDir) {
		t.Fatalf("expected false for empty dir %s", tmpDir)
	}

	// Adding flypy.dict.yaml
	flypyYaml := filepath.Join(tmpDir, "flypy.dict.yaml")
	if err := os.WriteFile(flypyYaml, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if !hasRimeDicts(tmpDir) {
		t.Fatalf("expected true when flypy.dict.yaml exists in %s", tmpDir)
	}
	_ = os.Remove(flypyYaml)

	// Adding flypy_dicts folder
	flypyDictsDir := filepath.Join(tmpDir, "flypy_dicts")
	if err := os.Mkdir(flypyDictsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if !hasRimeDicts(tmpDir) {
		t.Fatalf("expected true when flypy_dicts dir exists in %s", tmpDir)
	}
}

func TestDetectWeaselDeployer(t *testing.T) {
	path, ok := DetectWeaselDeployer()
	if runtime.GOOS == "windows" {
		if ok {
			t.Logf("Detected WeaselDeployer at: %s", path)
			if path == "" {
				t.Fatal("ok is true but path is empty")
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("detected deployer does not exist: %v", err)
			}
			if info.IsDir() {
				t.Fatalf("detected deployer is a directory: %s", path)
			}
		} else {
			t.Log("WeaselDeployer not detected on this Windows environment (expected in CI or fresh setup)")
		}
	} else {
		if ok || path != "" {
			t.Fatalf("expected (!ok, empty path) on non-windows platform, got ok=%v, path=%s", ok, path)
		}
	}
}

func TestDetectWeaselServer(t *testing.T) {
	path, exists, running := DetectWeaselServer()
	t.Logf("DetectWeaselServer: path=%q, exists=%v, running=%v", path, exists, running)
	if runtime.GOOS == "windows" {
		if exists {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				t.Fatalf("detected server path is invalid: %v", err)
			}
		}
	} else {
		if exists || running || path != "" {
			t.Fatalf("expected non-windows to return empty/false")
		}
	}
}

func TestTriggerDeploy_InvalidPath(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "non_existent_deployer.exe")
	err := TriggerDeploy(nonExistent)
	if err == nil {
		t.Fatal("expected error for non-existent deployer path, got nil")
	}
}

func TestTriggerDeploy_MockProcess(t *testing.T) {
	if runtime.GOOS != "windows" {
		err := TriggerDeploy("")
		if err == nil {
			t.Fatal("expected error on non-windows platform, got nil")
		}
		return
	}

	// Find go compiler to build mock executables
	goBin, err := exec.LookPath("go")
	if err != nil {
		goBin = filepath.Join(runtime.GOROOT(), "bin", "go.exe")
	}
	if _, err := os.Stat(goBin); err != nil {
		t.Skip("go compiler not found, skipping mock process compilation test")
	}

	tmpDir := t.TempDir()

	// 1. Mock deployer that succeeds when given /deploy
	successSrc := filepath.Join(tmpDir, "mock_success.go")
	successExe := filepath.Join(tmpDir, "mock_success.exe")
	srcContent := `package main

import "os"

func main() {
	hasDeploy := false
	for _, a := range os.Args[1:] {
		if a == "/deploy" {
			hasDeploy = true
			break
		}
	}
	if hasDeploy {
		os.Exit(0)
	}
	os.Exit(1)
}
`
	if err := os.WriteFile(successSrc, []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}

	buildCmd := exec.Command(goBin, "build", "-o", successExe, successSrc)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build mock success executable: %v, output: %s", err, string(out))
	}

	// Test successful deploy execution
	if err := TriggerDeploy(successExe); err != nil {
		t.Fatalf("expected TriggerDeploy to succeed with mock executable, got error: %v", err)
	}

	// 2. Mock deployer that fails (exits with non-zero)
	failSrc := filepath.Join(tmpDir, "mock_fail.go")
	failExe := filepath.Join(tmpDir, "mock_fail.exe")
	failSrcContent := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "mock deploy error")
	os.Exit(42)
}
`
	if err := os.WriteFile(failSrc, []byte(failSrcContent), 0644); err != nil {
		t.Fatal(err)
	}

	buildFailCmd := exec.Command(goBin, "build", "-o", failExe, failSrc)
	if out, err := buildFailCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build mock fail executable: %v, output: %s", err, string(out))
	}

	// Test failing deploy execution
	if err := TriggerDeploy(failExe); err == nil {
		t.Fatal("expected TriggerDeploy to fail with exit code 42, got nil error")
	}
}

func TestFileExistsHelper(t *testing.T) {
	tmpDir := t.TempDir()
	// Directory itself is not a regular file
	if fileExists(tmpDir) {
		t.Fatalf("expected fileExists to be false for directory %s", tmpDir)
	}

	// Non-existent file
	nonExistent := filepath.Join(tmpDir, "does_not_exist.txt")
	if fileExists(nonExistent) {
		t.Fatalf("expected fileExists to be false for non-existent file %s", nonExistent)
	}

	// Existing file
	existingFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(existingFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(existingFile) {
		t.Fatalf("expected fileExists to be true for file %s", existingFile)
	}
}
