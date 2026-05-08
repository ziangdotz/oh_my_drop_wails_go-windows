package main

import (
	"os"
	"path/filepath"
	"strings"
)

// 定义文件类型常量
const (
	TypeFolder = "folder"
	TypeZip    = "archive"
	TypeExe    = "executable"
	TypeImage  = "image"
	TypeFile   = "file"
)

func getFileType(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return TypeFile
	}
	if info.IsDir() {
		return TypeFolder
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip", ".rar", ".7z", ".tar", ".gz":
		return TypeZip
	case ".exe", ".msi", ".bat", ".sh", ".app":
		return TypeExe
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return TypeImage
	default:
		return TypeFile
	}
}
