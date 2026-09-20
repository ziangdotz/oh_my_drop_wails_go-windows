//go:build windows

package main

import (
	"context"
	"math"
	"os"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// 本文件是系统托盘的核心：窗口/消息循环、托盘图标注册、窗口过程（WndProc）、
// 弹出菜单的构建与命令分发。
// 相关的 Win32 声明、主题度量、菜单绘制与图标加载分别拆分至：
//   - systray_win32.go  常量、结构体与 DLL 过程
//   - systray_theme.go  配色、尺寸度量与 DPI 缩放
//   - systray_draw.go   菜单项自绘
//   - systray_icon.go   图标 embed 加载与缓存

var appCtx context.Context

var (
	trayWindow   windows.Handle
	trayIconData NOTIFYICONDATA
	menuItems    []trayMenuItem
)

type trayMenuItem struct {
	id   uint32
	text string
}

func initSystray(ctx context.Context) {
	appCtx = ctx
	go runTrayMessageLoop()
}

func runTrayMessageLoop() {
	hInstance, _, _ := getModuleHandle.Call(0)
	if hInstance == 0 {
		return
	}

	hIcon, _, _ := loadIcon.Call(0, uintptr(32512))
	if hIcon == 0 {
		return
	}

	customIcon := loadCustomIcon()
	if customIcon != 0 {
		hIcon = uintptr(customIcon)
	}

	className, _ := windows.UTF16PtrFromString(WNDS_CLASS_NAME)
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   windows.NewCallback(trayWndProc),
		HInstance:     windows.Handle(hInstance),
		HIcon:         windows.Handle(hIcon),
		HbrBackground: windows.Handle(0),
		LpszClassName: className,
		HIconSm:       windows.Handle(hIcon),
	}

	registerClass.Call(uintptr(unsafe.Pointer(&wc)))

	windowName, _ := windows.UTF16PtrFromString(WNDS_WINDOW_NAME)
	hwnd, _, _ := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0, 0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return
	}
	trayWindow = windows.Handle(hwnd)

	showWindow.Call(hwnd, 0)

	trayIconData = NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		HWnd:             trayWindow,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            windows.Handle(hIcon),
	}

	tip, _ := windows.UTF16FromString("ohMyDrop")
	copy(trayIconData.SzTip[:], tip)

	shellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&trayIconData)))

	var msg MSG
	for {
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		translateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func loadCustomIcon() windows.Handle {
	iconData := getEmbeddedIcon()
	if len(iconData) == 0 {
		return 0
	}

	tmpFile, err := os.CreateTemp("", "appicon_*.ico")
	if err != nil {
		return 0
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(iconData)
	tmpFile.Close()

	path, _ := windows.UTF16PtrFromString(tmpFile.Name())
	hIcon, _, _ := loadImage.Call(
		0,
		uintptr(unsafe.Pointer(path)),
		1,
		0, 0,
		0x00000010,
	)
	if hIcon == 0 {
		return 0
	}
	return windows.Handle(hIcon)
}

func getEmbeddedIcon() []byte {
	return embeddedIcon
}

func trayWndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONUP, WM_RBUTTONUP:
			showTrayMenu()
		}
	case WM_DRAWITEM:
		ds := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if ds.CtlType == ODT_MENU {
			id := uint32(ds.ItemID)
			if ds.ItemData != 0 {
				id = uint32(ds.ItemData)
			}
			text := ""
			if id != ID_SEPARATOR {
				text = findMenuTextByID(id)
			}
			drawMenuItem(windows.Handle(ds.HDC), ds.RcItem, ds.ItemState, id, text)
		}
		return 1
	case WM_MEASUREITEM:
		ms := (*MEASUREITEMSTRUCT)(unsafe.Pointer(lParam))
		if ms.CtlType == ODT_MENU {
			metrics := currentTrayMetrics()
			id := ms.ItemID
			if ms.ItemData != 0 {
				id = uint32(ms.ItemData)
			}

			if id == ID_SEPARATOR {
				ms.ItemWidth = uint32(metrics.baseWidth)
				ms.ItemHeight = uint32(metrics.separatorH)
			} else if id == ID_VERSION {
				ms.ItemWidth = uint32(metrics.baseWidth)
				ms.ItemHeight = uint32(metrics.versionH)
			} else {
				ms.ItemWidth = measureMenuItemWidth(findMenuTextByID(id), metrics)
				ms.ItemHeight = uint32(metrics.itemH)
			}
		}
		return 1
	case WND_DESTROY:
		shellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&trayIconData)))
		destroyWindow.Call(uintptr(hwnd))
		postQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := defWindowProc.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func showTrayMenu() {
	var pt POINT
	getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	palette := currentTrayPalette()

	hMenu, _, _ := createPopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer destroyMenu.Call(hMenu)

	customBgBrush, _, _ := createSolidBrush.Call(uintptr(palette.menuBG))
	defer deleteObject.Call(customBgBrush)
	mi := MENUINFO{
		CbSize:  uint32(unsafe.Sizeof(MENUINFO{})),
		FMask:   MIM_BACKGROUND | MIM_APPLYTOSUBMENUS,
		HbrBack: windows.Handle(customBgBrush),
	}
	setMenuInfo.Call(hMenu, uintptr(unsafe.Pointer(&mi)))

	menuItems = []trayMenuItem{
		{ID_VERSION, trayVersionMenuText()},
		{0, ""},
		{ID_SHOW, "显示窗口"},
		{ID_HIDE, "隐藏窗口"},
		{ID_THEME, themeToggleMenuText()},
		{0, ""},
		{ID_QUIT, "退出"},
	}

	for _, item := range menuItems {
		if item.id == 0 {
			addSeparator(windows.Handle(hMenu))
		} else {
			addOwnerDrawMenuItem(windows.Handle(hMenu), item.id, item.text)
		}
	}

	setForeground.Call(uintptr(trayWindow))

	cmd, _, _ := trackPopupMenu.Call(
		hMenu,
		TPM_LEFTALIGN|TPM_BOTTOMALIGN|TPM_RIGHTBUTTON|TPM_RETURNCMD,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		uintptr(trayWindow),
		0,
	)

	postMessage.Call(uintptr(trayWindow), WM_NULL, 0, 0)

	switch uint32(cmd) {
	case ID_SHOW:
		runtime.WindowShow(appCtx)
		runtime.WindowUnminimise(appCtx)
	case ID_HIDE:
		runtime.WindowHide(appCtx)
	case ID_THEME:
		nextTheme := toggleTrayMenuTheme()
		runtime.EventsEmit(appCtx, "theme_toggle", nextTheme)
	case ID_QUIT:
		runtime.Quit(appCtx)
	}
}

func addSeparator(hMenu windows.Handle) {
	mii := MENUITEMINFO{
		CbSize:     uint32(unsafe.Sizeof(MENUITEMINFO{})),
		FMask:      MIIM_ID | MIIM_FTYPE | MIIM_DATA,
		FType:      MFT_OWNERDRAW,
		WID:        ID_SEPARATOR,
		DwItemData: uintptr(ID_SEPARATOR),
	}
	insertMenuItem.Call(uintptr(hMenu), math.MaxUint32, 1, uintptr(unsafe.Pointer(&mii)))
}

func addOwnerDrawMenuItem(hMenu windows.Handle, id uint32, text string) {
	mii := MENUITEMINFO{
		CbSize:     uint32(unsafe.Sizeof(MENUITEMINFO{})),
		FMask:      MIIM_ID | MIIM_FTYPE | MIIM_DATA,
		FType:      MFT_OWNERDRAW,
		WID:        id,
		DwItemData: uintptr(id),
	}
	insertMenuItem.Call(uintptr(hMenu), math.MaxUint32, 1, uintptr(unsafe.Pointer(&mii)))
}

func findMenuTextByID(id uint32) string {
	for _, item := range menuItems {
		if item.id == id {
			return item.text
		}
	}
	return ""
}

func measureMenuItemWidth(text string, metrics trayMenuMetrics) uint32 {
	if text == "" {
		return uint32(metrics.baseWidth)
	}

	hdc, _, _ := getDC.Call(uintptr(trayWindow))
	if hdc == 0 {
		return uint32(metrics.baseWidth)
	}
	defer releaseDC.Call(uintptr(trayWindow), hdc)

	textPtr, _ := windows.UTF16PtrFromString(text)
	var size SIZE
	getTextExtent.Call(
		hdc,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(len([]rune(text))),
		uintptr(unsafe.Pointer(&size)),
	)

	textStartX := metrics.padding + metrics.iconLeft + metrics.iconSize + metrics.iconTextGap
	width := textStartX + size.Cx + metrics.padding
	if width < metrics.minWidth {
		width = metrics.minWidth
	}
	return uint32(width)
}
