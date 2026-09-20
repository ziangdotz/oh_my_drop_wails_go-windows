package main

import (
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// 本文件负责应用窗口的显隐与切换状态管理（含防抖），
// 与 Wails 运行时窗口 API 交互。

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
