package main

import (
	"embed"
	"flag"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	/* 定义并解析命令行参数 */
	startX := flag.Int("x", -1, "starting x position")
	startY := flag.Int("y", -1, "starting y position")
	flag.Parse()

	app := NewApp()
	app.initialX = *startX
	app.initialY = *startY

	appOptions := &options.App{
		Title:         "ohMyDrop",
		Width:         210,
		Height:        230,
		DisableResize: true,                           // 禁止调整窗口大小
		Frameless:     true,                           // 开启无边框模式
		AlwaysOnTop:   true,                           // 是否置顶
		StartHidden:   *startX == -1 && *startY == -1, // 启动时隐藏窗口：如果坐标是默认值 -1，说明是初次启动，开启隐藏；如果坐标不是 -1，说明是用户点击“新建窗口”启动的，直接显示
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,        // Win11 云母效果
			Messages:             &windows.Messages{}, // 禁用 Wails 默认的焦点获取逻辑，减少抢占
		},
		OnStartup: app.startup,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,                  // 开启文件原生拖拽
			DisableWebViewDrop: false,                 // 允许 Webview 接收
			CSSDropProperty:    "--wails-drop-target", // 默认属性
			CSSDropValue:       "drop",                // 默认值
		},
		Bind: []interface{}{
			app,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
	}

	err := wails.Run(appOptions)
	if err != nil {
		println("Error:", err.Error())
	}
}
