package main

import (
	"context"
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 绑定的核心结构体，承载应用运行状态；startup 为生命周期入口。
// 各具体职责按文件划分：
//   - hotkey.go        全局键盘钩子监听
//   - appwindow.go     窗口显隐与切换
//   - thumbnail.go     缩略图生成与缓存接入
//   - pathsecurity.go  拖放路径安全校验
//   - fileoperation.go 文件拖放操作调度

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
