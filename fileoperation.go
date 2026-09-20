package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// 本文件负责文件拖放操作的生命周期管理：标记操作类型、TTL 过期惰性清理、
// 完成时批量校验路径并调用平台剪贴板实现（writeFilesToClipboard 见 clipboard_windows.go）。

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// 标记文件操作（前端 DragStart 调用）
func (a *App) HandleFilePaths_MarkFileOperate(operationType string) {
	a.mu.Lock()
	a.fileOperateType = operationType
	a.mu.Unlock()
}

// 读取文件操作类型（加锁读取到局部变量，避免持锁执行耗时调用）
func (a *App) getFileOperateType() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.fileOperateType
}

// fileOperation 记录一次待完成的文件拖放操作及其创建时间，用于 TTL 过期清理。
type fileOperation struct {
	path      string
	createdAt time.Time
}

// fileOperationTTL 为未完成操作的最长保留时间，超过则视为失效并在下次操作时惰性清理。
const fileOperationTTL = 5 * time.Minute

// sweepExpiredOperations 清理超过 TTL 仍未完成的文件操作条目，防止 sync.Map 无限增长。
func (a *App) sweepExpiredOperations() {
	now := time.Now()
	a.fileOperations.Range(func(key, value any) bool {
		if op, ok := value.(fileOperation); ok && now.Sub(op.createdAt) > fileOperationTTL {
			a.fileOperations.Delete(key)
		}
		return true
	})
}

// 标记文件路径（前端 DragStart 调用）
func (a *App) HandleFilePaths_MarkFileOperation(filePath string) string {
	if !isValidPath(filePath) {
		return ""
	}
	a.sweepExpiredOperations()                                                                // 惰性清理过期条目
	operationID := "fileoperations_" + filepath.Base(filePath) + "_" + uuid()                 // 生成唯一操作 ID
	a.fileOperations.Store(operationID, fileOperation{path: filePath, createdAt: time.Now()}) // 存储操作状态
	return operationID
}

// 执行文件操作（前端 DragEnd 调用）
func (a *App) HandleFilePaths_CompleteOperation(operationIDs []string) string {
	a.sweepExpiredOperations() // 惰性清理过期条目
	/* 批量验证并收集路径 */
	var validPaths []string
	for _, id := range operationIDs {
		rawPath, exists := a.fileOperations.Load(id)
		if !exists {
			continue
		}
		data, ok := rawPath.(fileOperation)
		if !ok {
			continue
		}
		originalPath := data.path
		if !isValidPath(originalPath) {
			continue
		}
		if _, err := os.Stat(originalPath); os.IsNotExist(err) {
			a.fileOperations.Delete(id) // 清理文件不存在的失效 ID
			continue
		}
		validPaths = append(validPaths, originalPath)
	}
	if len(validPaths) == 0 {
		return "[Error]: 没有有效的文件可操作"
	}

	/* 调用对应平台的原生实现 */
	operateType := a.getFileOperateType()
	isMove := operateType == "move"
	err := a.writeFilesToClipboard(validPaths, isMove)
	if err != nil {
		return "[Error]: 剪切板操作失败"
	}

	/* 统一清理操作状态 */
	for _, id := range operationIDs {
		a.fileOperations.Delete(id)
	}
	return "success"
}

/* 操作剪切板 */
func (a *App) HandleFilePaths_OperateWithClipboard(filePaths []string) string {
	/* 批量验证并收集路径 */
	var validPaths []string
	for _, filePath := range filePaths {
		if !isValidPath(filePath) {
			continue
		}
		validPaths = append(validPaths, filePath)
	}
	if len(validPaths) == 0 {
		return "[Error]: 没有有效的文件可操作"
	}

	/* 调用对应平台的原生实现 */
	operateType := a.getFileOperateType()
	isMove := operateType == "move"
	err := a.writeFilesToClipboard(validPaths, isMove)
	if err != nil {
		return "[Error]: 剪切板操作失败"
	}
	return "success"
}
