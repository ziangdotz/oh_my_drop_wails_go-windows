package main

import (
	"os"
	"path/filepath"
	"strings"
)

// 本文件负责拖放路径的安全校验：拒绝系统/程序安装/用户 AppData 等敏感目录，
// 防止危险路径成为文件操作目标。

// 路径安全检查
func isValidPath(path string) bool {
	cleanPath := filepath.Clean(path)
	/* 必须是绝对路径 */
	if !filepath.IsAbs(cleanPath) {
		return false
	}
	/* 敏感目录黑名单 (转为小写比较，适应 Windows 不区分大小写) */
	lowerPath := strings.ToLower(cleanPath)
	sep := strings.ToLower(string(os.PathSeparator))
	/* 系统/程序安装/数据目录，均禁止作为拖放操作目标 */
	blacklistEnv := []string{
		"SystemRoot", "windir",
		"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432",
		"ProgramData",
	}
	for _, key := range blacklistEnv {
		dir := os.Getenv(key)
		if dir == "" {
			continue
		}
		if inDir(lowerPath, strings.ToLower(filepath.Clean(dir)), sep) {
			return false
		}
	}
	/* 用户 AppData 目录（含漫游/本地应用数据） */
	if profile := os.Getenv("USERPROFILE"); profile != "" {
		appData := strings.ToLower(filepath.Clean(filepath.Join(profile, "AppData")))
		if inDir(lowerPath, appData, sep) {
			return false
		}
	}
	return true
}

// inDir 判断 lowerPath 是否位于 dir 目录内（含自身），带分隔符边界检查，
// 避免 "c:\\windows" 误匹配 "c:\\windowsfoo" 这类前缀。
func inDir(lowerPath, dir, sep string) bool {
	if dir == "" {
		return false
	}
	return lowerPath == dir || strings.HasPrefix(lowerPath, dir+sep)
}

// 检查文件是否可用
func (a *App) CheckFileExists(filePath string) bool {
	if !isValidPath(filePath) {
		return false
	}
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir()
}
