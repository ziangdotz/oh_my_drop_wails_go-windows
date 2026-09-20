package main

import (
	hook "github.com/robotn/gohook"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// 本文件负责全局键盘钩子监听：基于 gohook 捕获物理按键，
// 实现「右 Alt 触发窗口显隐」「左 Alt + Q 退出程序」等全局快捷键。

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
