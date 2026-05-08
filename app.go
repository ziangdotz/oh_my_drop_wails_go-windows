package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	hook "github.com/robotn/gohook"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx               context.Context
	initialX          int
	initialY          int
	isRightAltPressed bool       // 物理按键锁 右 Alt
	isLeftAltPressed  bool       // 物理按键锁 左 Ctrl
	isTriggered       bool       // 用于防止长按时重复触发业务逻辑
	isVisible         bool       // 内部追踪窗口状态
	mu                sync.Mutex // 添加互斥锁
	fileOperateType   string
	fileOperations    sync.Map
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	initSystray(ctx)

	wailsRuntime.WindowCenter(ctx)
	a.isVisible = false
	go a.listenGlobalSpace() // 启动监听
}

func (a *App) listenGlobalSpace() {
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		/* 识别并更新 左 Alt（164）的状态 */
		if ev.Rawcode == 164 {
			if ev.Kind == hook.KeyDown {
				a.isLeftAltPressed = true
			} else if ev.Kind == hook.KeyUp {
				a.isLeftAltPressed = false
			}
		}
		/* 当左 Alt 按住时，按下 Q（81）则退出程序 */
		if ev.Kind == hook.KeyDown && ev.Rawcode == 81 {
			if a.isLeftAltPressed {
				wailsRuntime.Quit(a.ctx)
				return
			}
		}

		/* 识别并更新 Right Alt（165）的状态 */
		if ev.Rawcode == 165 {
			if ev.Kind == hook.KeyDown {
				a.isRightAltPressed = true
			} else if ev.Kind == hook.KeyUp {
				a.isRightAltPressed = false
				a.isTriggered = false // 只要松开其中一个，就重置触发标记
			}
		}

		if a.isRightAltPressed {
			/* 如果已经触发过，且按键没松开，直接跳过（防抖） */
			if a.isTriggered {
				continue
			}
			/* 触发后立即锁定，防止长按连发 */
			a.isTriggered = true
			/* 显示窗口 */
			go a.togglePreview()
		}
	}
}

var lastToggle time.Time

func (a *App) togglePreview() {
	a.mu.Lock()
	if time.Since(lastToggle) < 200*time.Millisecond {
		a.mu.Unlock()
		return
	}
	lastToggle = time.Now()
	defer a.mu.Unlock()

	if a.isVisible {
		a.isVisible = false
		wailsRuntime.WindowHide(a.ctx)
	} else {
		a.isVisible = true // 先设置状态
		go func() {
			wailsRuntime.WindowShow(a.ctx)
			wailsRuntime.WindowUnminimise(a.ctx)
			if a.initialX == -1 && a.initialY == -1 {
				time.Sleep(50 * time.Millisecond)
				wailsRuntime.WindowCenter(a.ctx)
			}
		}()
	}
}

func (a *App) HideWindow() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isVisible = false
	wailsRuntime.WindowHide(a.ctx)
}

type Thumbnail struct {
	Path        string  `json:"path"`
	Base64      string  `json:"base64"`
	Ratio       float64 `json:"ratio"`       // 宽度 / 高度
	Orientation string  `json:"orientation"` // landscape 或 portrait
}

var thumbnailsCache sync.Map

// 生成缩略图
func (a *App) HandleFilePaths_GetThumbnails(paths []string) {
	sem := make(chan struct{}, 5) // 限制最大并发处理数为 5，防止 CPU 和内存瞬间过载
	var wg sync.WaitGroup

	for _, path := range paths {
		/* 检查缓存 */
		if val, ok := thumbnailsCache.Load(path); ok {
			wailsRuntime.EventsEmit(a.ctx, "thumbnails_ready", val)
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		/* 开启协程并行处理每一张图片 */
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量
			fileType := getFileType(p)
			var data Thumbnail
			if fileType == TypeImage {
				// 1. 打开图片并自动处理 EXIF 旋转方向
				src, err := imaging.Open(path, imaging.AutoOrientation(true))
				if err != nil {
					return
				}
				// 2. 生成缩略图：高度固定 100，宽度设为 0 表示等比例缩放
				thumb := imaging.Resize(src, 0, 100, imaging.Linear)
				// 编码为 JPEG Base64
				buf := new(bytes.Buffer)
				err = jpeg.Encode(buf, thumb, &jpeg.Options{Quality: 50})
				if err != nil {
					return
				}
				// 4. 获取原始尺寸并计算精确比例
				bounds := src.Bounds()
				w, h := float64(bounds.Dx()), float64(bounds.Dy())
				ratio := w / h
				// 5. 判定横竖屏状态
				orientation := "portrait"
				if ratio > 1 {
					orientation = "landscape"
				}

				data = Thumbnail{
					Path:        p,
					Base64:      fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes())),
					Ratio:       ratio,
					Orientation: orientation,
				}
			} else {
				icon := predefinedIcons[fileType]
				orientation := "portrait"
				if icon.Ratio > 1 {
					orientation = "landscape"
				}
				data = Thumbnail{
					Path:        p,
					Base64:      icon.Base64,
					Ratio:       icon.Ratio,
					Orientation: orientation,
				}
			}

			thumbnailsCache.Store(p, data) // 存入缓存
			wailsRuntime.EventsEmit(a.ctx, "thumbnails_ready", data)
		}(path)
	}
	wg.Wait() // 等待所有任务完成
}

// 清除缩略图缓存
func (a *App) HandleFilePaths_ClearThumbnailsCache() {
	thumbnailsCache = sync.Map{}
}

// 路径安全检查
func isValidPath(path string) bool {
	cleanPath := filepath.Clean(path)
	/* 必须是绝对路径 */
	if !filepath.IsAbs(cleanPath) {
		return false
	}
	/* 敏感目录黑名单 (转为小写比较，适应 Windows 不区分大小写) */
	lowerPath := strings.ToLower(cleanPath)
	blacklist := []string{
		strings.ToLower(os.Getenv("SystemRoot")),
	}
	for _, prefix := range blacklist {
		if prefix != "" && strings.HasPrefix(lowerPath, prefix) {
			return false
		}
	}
	return true
}

// 检查文件是否可用
func (a *App) CheckFileExists(filePath string) bool {
	if !isValidPath(filePath) {
		return false
	}
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir()
}

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// 标记文件操作（前端 DragStart 调用）
func (a *App) HandleFilePaths_MarkFileOperate(operationType string) {
	a.fileOperateType = operationType
}

// 标记文件路径（前端 DragStart 调用）
func (a *App) HandleFilePaths_MarkFileOperation(filePath string) string {
	if !isValidPath(filePath) {
		return ""
	}
	operationID := "fileoperations_" + filepath.Base(filePath) + "_" + uuid() // 生成唯一操作 ID
	a.fileOperations.Store(operationID, map[string]string{"path": filePath})  // 存储操作状态
	return operationID
}

// 执行文件操作（前端 DragEnd 调用）
func (a *App) HandleFilePaths_CompleteOperation(operationIDs []string) string {
	/* 批量验证并收集路径 */
	var validPaths []string
	for _, id := range operationIDs {
		rawPath, exists := a.fileOperations.Load(id)
		if !exists {
			continue
		}
		data, ok := rawPath.(map[string]string)
		if !ok {
			continue
		}
		originalPath := data["path"]
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
	isMove := a.fileOperateType == "move"
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
	isMove := a.fileOperateType == "move"
	err := a.writeFilesToClipboard(validPaths, isMove)
	if err != nil {
		return "[Error]: 剪切板操作失败"
	}
	return "success"
}
