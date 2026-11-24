package rime

import (
	"os"
	"path/filepath"
)

// fileExists reports whether the given path exists and is a regular file (not a directory).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// hasRimeDicts reports whether the directory contains flypy.dict.yaml or a flypy_dicts folder.
func hasRimeDicts(dir string) bool {
	valid, _ := ValidateRimeDir(dir)
	return valid
}

// ValidateRimeDir checks whether the specified directory is a valid Rime configuration directory
// containing the rime-xhkk schema or dictionary files.
// Returns (true, "") if valid, or (false, reason) if invalid.
func ValidateRimeDir(dir string) (bool, string) {
	dir = filepath.Clean(dir)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false, "目录不存在或不是文件夹"
	}

	// Check for core flypy dictionary or schema files
	if fileExists(filepath.Join(dir, "flypy.dict.yaml")) {
		return true, ""
	}
	if fileExists(filepath.Join(dir, "flypy.schema.yaml")) {
		return true, ""
	}
	if dInfo, err := os.Stat(filepath.Join(dir, "flypy_dicts")); err == nil && dInfo.IsDir() {
		return true, ""
	}
	if fileExists(filepath.Join(dir, "default.yaml")) || fileExists(filepath.Join(dir, "default.custom.yaml")) {
		return false, "该目录包含 Rime 配置，但缺少小鹤音形方案文件 (flypy.dict.yaml)"
	}
	return false, "无效目录：未找到小鹤音形配置文件 (flypy.dict.yaml 或 flypy_dicts/)"
}
