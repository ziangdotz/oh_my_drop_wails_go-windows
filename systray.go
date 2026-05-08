package main

import (
	"context"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

var appCtx context.Context

const (
	WM_USER             = 0x0400
	WNDS_CLASS_NAME     = "OhMyDropTrayClass"
	WNDS_WINDOW_NAME    = "OhMyDropTrayWindow"
	WM_TRAYICON         = WM_USER + 1
	WM_LBUTTONUP        = 0x0202
	WM_RBUTTONUP        = 0x0205
	WM_DRAWITEM         = 0x002B
	WM_MEASUREITEM      = 0x002C
	WM_NULL             = 0x0000
	NIM_ADD             = 0x00000000
	NIM_DELETE          = 0x00000002
	NIF_MESSAGE         = 0x00000001
	NIF_ICON            = 0x00000002
	NIF_TIP             = 0x00000004
	TPM_LEFTALIGN       = 0x0000
	TPM_BOTTOMALIGN     = 0x0020
	TPM_RIGHTBUTTON     = 0x0002
	TPM_RETURNCMD       = 0x0100
	ID_VERSION          = 7
	ID_SHOW             = 1
	ID_HIDE             = 2
	ID_SETTING          = 3
	ID_THEME            = 4
	ID_SEPARATOR        = 5
	ID_QUIT             = 6
	WND_DESTROY         = 0x0002
	MIM_BACKGROUND      = 0x00000002
	MIM_APPLYTOSUBMENUS = 0x80000000
	ODT_MENU            = 1
	ODS_SELECTED        = 0x0001
	MIIM_ID             = 0x00000002
	MIIM_FTYPE          = 0x00000100
	MIIM_DATA           = 0x00000020
	MFT_OWNERDRAW       = 0x00000100
	MENU_PADDING        = 7 // 菜单左右间距
	MENU_VERSION_H      = 24
	MENU_ITEM_HEIGHT    = 30
	MENU_CORNER_RADIUS  = 6
	MENU_SEPARATOR_H    = 6
	MENU_ICON_SIZE      = 12
	MENU_ICON_LEFT      = 7
	MENU_ICON_TEXT_GAP  = 5
)

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             windows.Handle
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

type POINT struct {
	X int32
	Y int32
}

type MENUITEMINFO struct {
	CbSize        uint32
	FMask         uint32
	FType         uint32
	FState        uint32
	WID           uint32
	HSubMenu      windows.Handle
	HbmpChecked   windows.Handle
	HbmpUnchecked windows.Handle
	DwItemData    uintptr
	DwTypeData    *uint16
	Cch           uint32
	HbmpItem      windows.Handle
}

type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type MENUINFO struct {
	CbSize          uint32
	FMask           uint32
	DwStyle         uint32
	CyMax           uint32
	HbrBack         windows.Handle
	DwContextHelpID uint32
	DwMenuData      uintptr
}

var (
	shell32         = windows.NewLazySystemDLL("shell32.dll")
	trayUser32      = windows.NewLazySystemDLL("user32.dll")
	trayKernel32    = windows.NewLazySystemDLL("kernel32.dll")
	gdi32           = windows.NewLazySystemDLL("gdi32.dll")
	shellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")

	createWindowEx   = trayUser32.NewProc("CreateWindowExW")
	defWindowProc    = trayUser32.NewProc("DefWindowProcW")
	destroyWindow    = trayUser32.NewProc("DestroyWindow")
	dispatchMessage  = trayUser32.NewProc("DispatchMessageW")
	getCursorPos     = trayUser32.NewProc("GetCursorPos")
	getMessage       = trayUser32.NewProc("GetMessageW")
	getModuleHandle  = trayKernel32.NewProc("GetModuleHandleW")
	loadIcon         = trayUser32.NewProc("LoadIconW")
	loadImage        = trayUser32.NewProc("LoadImageW")
	postQuitMessage  = trayUser32.NewProc("PostQuitMessage")
	registerClass    = trayUser32.NewProc("RegisterClassExW")
	setForeground    = trayUser32.NewProc("SetForegroundWindow")
	showWindow       = trayUser32.NewProc("ShowWindow")
	trackPopupMenu   = trayUser32.NewProc("TrackPopupMenu")
	postMessage      = trayUser32.NewProc("PostMessageW")
	translateMsg     = trayUser32.NewProc("TranslateMessage")
	createPopupMenu  = trayUser32.NewProc("CreatePopupMenu")
	insertMenuItem   = trayUser32.NewProc("InsertMenuItemW")
	destroyMenu      = trayUser32.NewProc("DestroyMenu")
	getDC            = trayUser32.NewProc("GetDC")
	releaseDC        = trayUser32.NewProc("ReleaseDC")
	selectObject     = gdi32.NewProc("SelectObject")
	deleteObject     = gdi32.NewProc("DeleteObject")
	getTextExtent    = gdi32.NewProc("GetTextExtentPoint32W")
	setMenuInfo      = trayUser32.NewProc("SetMenuInfo")
	getStockObject   = gdi32.NewProc("GetStockObject")
	fillRect         = trayUser32.NewProc("FillRect")
	createSolidBrush = gdi32.NewProc("CreateSolidBrush")
	roundRect        = gdi32.NewProc("RoundRect")
	rectangle        = gdi32.NewProc("Rectangle")
	ellipse          = gdi32.NewProc("Ellipse")
	setTextColor     = gdi32.NewProc("SetTextColor")
	setBkMode        = gdi32.NewProc("SetBkMode")
	drawText         = trayUser32.NewProc("DrawTextW")
	moveToEx         = gdi32.NewProc("MoveToEx")
	lineTo           = gdi32.NewProc("LineTo")
	createPen        = gdi32.NewProc("CreatePen")
	setPixelV        = gdi32.NewProc("SetPixelV")
	getDpiForWindow  = trayUser32.NewProc("GetDpiForWindow")
)

var (
	trayWindow    windows.Handle
	trayIconData  NOTIFYICONDATA
	menuItems     []trayMenuItem
	trayMenuTheme = "dark"
	trayMenuIcons = map[string]*trayIconPixels{}
)

type trayIconPixels struct {
	width  int
	height int
	pix    []uint8 // NRGBA pixels
}

type trayMenuPalette struct {
	menuBG         uint32
	itemBG         uint32
	itemBGSelected uint32
	separator      uint32
	textColor      uint32
	iconColor      uint32
	iconColorSel   uint32
	iconVariant    string
}

type trayMenuItem struct {
	id   uint32
	text string
}

type trayMenuMetrics struct {
	padding      int32
	versionH     int32
	itemH        int32
	cornerRadius int32
	separatorH   int32
	iconSize     int32
	iconLeft     int32
	iconTextGap  int32
	lineInset    int32
	minWidth     int32
	baseWidth    int32
	rightTextPad int32
}

func trayVersionMenuText() string {
	return "v1.0.1"
}

func currentTrayPalette() trayMenuPalette {
	if trayMenuTheme == "light" {
		return trayMenuPalette{
			menuBG:         0xF7F3EE,
			itemBG:         0xF7F3EE,
			itemBGSelected: 0xEDE6DE,
			separator:      0xD5CCC2,
			textColor:      0x1D1D1D,
			iconColor:      0x303030,
			iconColorSel:   0x1D1D1D,
			iconVariant:    "dark",
		}
	}

	return trayMenuPalette{
		menuBG:         0x1A1A1A,
		itemBG:         0x1A1A1A,
		itemBGSelected: 0x2E2E2E,
		separator:      0x3C3C3C,
		textColor:      0xEEEEEE,
		iconColor:      0xBEBEBE,
		iconColorSel:   0xFFFFFF,
		iconVariant:    "light",
	}
}

func toggleTrayMenuTheme() string {
	if trayMenuTheme == "dark" {
		trayMenuTheme = "light"
	} else {
		trayMenuTheme = "dark"
	}
	return trayMenuTheme
}

func themeToggleMenuText() string {
	if trayMenuTheme == "light" {
		return "切换到深色"
	}
	return "切换到浅色"
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     int32
	ItemAction uint32
	ItemState  uint32
	HwndItem   windows.Handle
	HDC        windows.Handle
	RcItem     RECT
	ItemData   uintptr
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type SIZE struct {
	Cx int32
	Cy int32
}

type MEASUREITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemWidth  uint32
	ItemHeight uint32
	ItemData   uintptr
}

func initSystray(ctx context.Context) {
	appCtx = ctx
	go runTrayMessageLoop()
}

func currentTrayScale() float64 {
	if getDpiForWindow.Find() != nil || trayWindow == 0 {
		return 1.0
	}

	dpi, _, _ := getDpiForWindow.Call(uintptr(trayWindow))
	if dpi == 0 {
		return 1.0
	}

	scale := float64(dpi) / 96.0
	if scale < 1.0 {
		return 1.0
	}
	return scale
}

func scaleMetric(base int32, scale float64) int32 {
	v := int32(math.Round(float64(base) * scale))
	if v < 1 {
		return 1
	}
	return v
}

func currentTrayMetrics() trayMenuMetrics {
	scale := currentTrayScale()
	return trayMenuMetrics{
		padding:      scaleMetric(MENU_PADDING, scale),
		versionH:     scaleMetric(MENU_VERSION_H, scale),
		itemH:        scaleMetric(MENU_ITEM_HEIGHT, scale),
		cornerRadius: scaleMetric(MENU_CORNER_RADIUS, scale),
		separatorH:   scaleMetric(MENU_SEPARATOR_H, scale),
		iconSize:     scaleMetric(MENU_ICON_SIZE, scale),
		iconLeft:     scaleMetric(MENU_ICON_LEFT, scale),
		iconTextGap:  scaleMetric(MENU_ICON_TEXT_GAP, scale),
		lineInset:    scaleMetric(2, scale),
		minWidth:     scaleMetric(84, scale),
		baseWidth:    scaleMetric(108, scale),
		rightTextPad: scaleMetric(2, scale),
	}
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

func drawMenuItem(hdc windows.Handle, rcItem RECT, itemState uint32, id uint32, text string) {
	palette := currentTrayPalette()
	metrics := currentTrayMetrics()
	fullBlackBrush, _, _ := createSolidBrush.Call(uintptr(palette.menuBG))
	if fullBlackBrush != 0 {
		fillRect.Call(
			uintptr(hdc),
			uintptr(unsafe.Pointer(&rcItem)),
			fullBlackBrush,
		)
		deleteObject.Call(fullBlackBrush)
	}

	if text == "" {
		pen, _, _ := createPen.Call(0, 1, uintptr(palette.separator))
		if pen != 0 {
			oldPen, _, _ := selectObject.Call(uintptr(hdc), pen)
			y := uintptr((rcItem.Top + rcItem.Bottom) / 2)
			startX := uintptr(rcItem.Left + metrics.padding + metrics.lineInset)
			endX := uintptr(rcItem.Right - metrics.padding - metrics.lineInset)
			moveToEx.Call(uintptr(hdc), startX, y, 0)
			lineTo.Call(uintptr(hdc), endX, y)
			if oldPen != 0 {
				selectObject.Call(uintptr(hdc), oldPen)
			}
			deleteObject.Call(pen)
		}
		return
	}

	if id == ID_VERSION {
		setTextColor.Call(uintptr(hdc), uintptr(palette.iconColor))
		setBkMode.Call(uintptr(hdc), 1)
		textPtr, _ := windows.UTF16PtrFromString(text)
		versionRect := RECT{
			Left:   rcItem.Left + metrics.padding,
			Top:    rcItem.Top,
			Right:  rcItem.Right - metrics.padding,
			Bottom: rcItem.Bottom,
		}
		drawText.Call(
			uintptr(hdc),
			uintptr(unsafe.Pointer(textPtr)),
			^uintptr(0),
			uintptr(unsafe.Pointer(&versionRect)),
			0x00000020|0x00000004|0x00000001,
		)
		return
	}

	itemBgColor := uintptr(palette.itemBG)
	if itemState&ODS_SELECTED != 0 {
		itemBgColor = uintptr(palette.itemBGSelected)
	}

	itemRect := RECT{
		Left:   rcItem.Left + metrics.padding,
		Top:    rcItem.Top,
		Right:  rcItem.Right - metrics.padding,
		Bottom: rcItem.Bottom,
	}
	itemBrush, _, _ := createSolidBrush.Call(itemBgColor)
	if itemBrush != 0 {
		nullPen, _, _ := getStockObject.Call(8) // 获取系统内置的 NULL_PEN（无边框画笔：常量值通常是 8）
		oldBrush, _, _ := selectObject.Call(uintptr(hdc), itemBrush)
		oldPen, _, _ := selectObject.Call(uintptr(hdc), nullPen) // 禁用边框线
		roundRect.Call(
			uintptr(hdc),
			uintptr(itemRect.Left),
			uintptr(itemRect.Top),
			uintptr(itemRect.Right),
			uintptr(itemRect.Bottom),
			uintptr(metrics.cornerRadius),
			uintptr(metrics.cornerRadius),
		)
		if oldBrush != 0 {
			selectObject.Call(uintptr(hdc), oldBrush)
		}
		if oldPen != 0 {
			selectObject.Call(uintptr(hdc), oldPen)
		}
		deleteObject.Call(itemBrush)
	}

	drawMenuIcon(hdc, itemRect, id, itemState&ODS_SELECTED != 0)

	setTextColor.Call(uintptr(hdc), uintptr(palette.textColor))
	setBkMode.Call(uintptr(hdc), 1)

	textPtr, _ := windows.UTF16PtrFromString(text)
	textRect := RECT{
		Left:   itemRect.Left + metrics.iconLeft + metrics.iconSize + metrics.iconTextGap,
		Top:    itemRect.Top,
		Right:  itemRect.Right - metrics.rightTextPad, // 文字右边留出间距
		Bottom: itemRect.Bottom,
	}
	drawText.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(textPtr)),
		^uintptr(0),
		uintptr(unsafe.Pointer(&textRect)),
		0x00000020|0x00000004, // 菜单对齐方式：DT_VCENTER（垂直居中） | DT_SINGLELINE（左对齐）
	)
}

func drawMenuIcon(hdc windows.Handle, itemRect RECT, id uint32, isSelected bool) {
	if id == ID_SEPARATOR || id == ID_VERSION || id == 0 {
		return
	}
	metrics := currentTrayMetrics()

	if drawMenuIconPNG(hdc, itemRect, id, isSelected) {
		return
	}

	iconLeft := itemRect.Left + metrics.iconLeft
	iconTop := (itemRect.Top + itemRect.Bottom - metrics.iconSize) / 2
	iconRight := iconLeft + metrics.iconSize
	iconBottom := iconTop + metrics.iconSize
	s := metrics.iconSize
	boxLeft := iconLeft + 1
	boxTop := iconTop + 1
	boxRight := iconRight - 1
	boxBottom := iconBottom - 1

	palette := currentTrayPalette()
	iconColor := uintptr(palette.iconColor)
	if isSelected {
		iconColor = uintptr(palette.iconColorSel)
	}

	penWidth := uintptr(2)
	if metrics.iconSize >= 20 {
		penWidth = 3
	}
	pen, _, _ := createPen.Call(0, penWidth, iconColor)
	if pen == 0 {
		return
	}
	defer deleteObject.Call(pen)

	oldPen, _, _ := selectObject.Call(uintptr(hdc), pen)
	nullBrush, _, _ := getStockObject.Call(5) // NULL_BRUSH
	oldBrush, _, _ := selectObject.Call(uintptr(hdc), nullBrush)

	defer func() {
		if oldPen != 0 {
			selectObject.Call(uintptr(hdc), oldPen)
		}
		if oldBrush != 0 {
			selectObject.Call(uintptr(hdc), oldBrush)
		}
	}()

	switch id {
	case ID_SHOW:
		winLeft := iconLeft + 1
		winTop := iconTop + 3
		winRight := iconRight - s/3
		winBottom := iconBottom - 1
		rectangle.Call(uintptr(hdc), uintptr(winLeft), uintptr(winTop), uintptr(winRight), uintptr(winBottom))
		ax1 := iconLeft + s/3
		ay1 := iconTop + (s*2)/3
		ax2 := iconRight - 2
		ay2 := iconTop + 2
		head := s / 4
		moveToEx.Call(uintptr(hdc), uintptr(ax1), uintptr(ay1), 0)
		lineTo.Call(uintptr(hdc), uintptr(ax2), uintptr(ay2))
		moveToEx.Call(uintptr(hdc), uintptr(ax2-head), uintptr(ay2), 0)
		lineTo.Call(uintptr(hdc), uintptr(ax2), uintptr(ay2))
		lineTo.Call(uintptr(hdc), uintptr(ax2), uintptr(ay2+head))
		// Add a subtle title bar line to improve visual precision.
		moveToEx.Call(uintptr(hdc), uintptr(winLeft+2), uintptr(winTop+2), 0)
		lineTo.Call(uintptr(hdc), uintptr(winRight-2), uintptr(winTop+2))
	case ID_HIDE:
		rectangle.Call(uintptr(hdc), uintptr(iconLeft+1), uintptr(iconTop+3), uintptr(iconRight-1), uintptr(iconBottom-1))
		lineY := iconTop + (s*3)/4
		moveToEx.Call(uintptr(hdc), uintptr(iconLeft+s/4), uintptr(lineY), 0)
		lineTo.Call(uintptr(hdc), uintptr(iconRight-s/4), uintptr(lineY))
	case ID_THEME: // ID_SETTING 暂时移除
		cx := iconLeft + s/2
		cy := iconTop + s/2
		outer := s/2 - 1
		inner := s/2 - 5
		ellipse.Call(uintptr(hdc), uintptr(cx-inner), uintptr(cy-inner), uintptr(cx+inner), uintptr(cy+inner))
		// 8-direction short spokes to simulate a finer gear.
		moveToEx.Call(uintptr(hdc), uintptr(cx), uintptr(cy-outer), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx), uintptr(cy-inner-1))
		moveToEx.Call(uintptr(hdc), uintptr(cx), uintptr(cy+inner+1), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx), uintptr(cy+outer))
		moveToEx.Call(uintptr(hdc), uintptr(cx-outer), uintptr(cy), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx-inner-1), uintptr(cy))
		moveToEx.Call(uintptr(hdc), uintptr(cx+inner+1), uintptr(cy), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx+outer), uintptr(cy))
		diagOuter := outer - 1
		diagInner := inner + 1
		moveToEx.Call(uintptr(hdc), uintptr(cx-diagOuter), uintptr(cy-diagOuter), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx-diagInner), uintptr(cy-diagInner))
		moveToEx.Call(uintptr(hdc), uintptr(cx+diagInner), uintptr(cy+diagInner), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx+diagOuter), uintptr(cy+diagOuter))
		moveToEx.Call(uintptr(hdc), uintptr(cx+diagInner), uintptr(cy-diagInner), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx+diagOuter), uintptr(cy-diagOuter))
		moveToEx.Call(uintptr(hdc), uintptr(cx-diagOuter), uintptr(cy+diagOuter), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx-diagInner), uintptr(cy+diagInner))
	case ID_QUIT:
		cx := iconLeft + s/2
		ellipse.Call(uintptr(hdc), uintptr(boxLeft), uintptr(boxTop), uintptr(boxRight), uintptr(boxBottom))
		moveToEx.Call(uintptr(hdc), uintptr(cx), uintptr(iconTop+1), 0)
		lineTo.Call(uintptr(hdc), uintptr(cx), uintptr(iconTop+s/2))
		// Carve a small top gap with background color to make a cleaner power symbol.
		gapColor := uintptr(palette.itemBG)
		if isSelected {
			gapColor = uintptr(palette.itemBGSelected)
		}
		gapPen, _, _ := createPen.Call(0, penWidth+1, gapColor)
		if gapPen != 0 {
			oldGapPen, _, _ := selectObject.Call(uintptr(hdc), gapPen)
			moveToEx.Call(uintptr(hdc), uintptr(cx-s/6), uintptr(iconTop+1), 0)
			lineTo.Call(uintptr(hdc), uintptr(cx+s/6), uintptr(iconTop+1))
			if oldGapPen != 0 {
				selectObject.Call(uintptr(hdc), oldGapPen)
			}
			deleteObject.Call(gapPen)
		}
	}
}

func drawMenuIconPNG(hdc windows.Handle, itemRect RECT, id uint32, isSelected bool) bool {
	icon := getTrayMenuIconByID(id)
	if icon == nil || icon.width <= 0 || icon.height <= 0 {
		return false
	}
	metrics := currentTrayMetrics()

	size := int(metrics.iconSize)
	if size <= 0 {
		return false
	}

	iconLeft := int(itemRect.Left + metrics.iconLeft)
	iconTop := int((itemRect.Top + itemRect.Bottom - metrics.iconSize) / 2)

	palette := currentTrayPalette()
	bg := palette.itemBG
	if isSelected {
		bg = palette.itemBGSelected
	}
	bgR, bgG, bgB := uint32(bg&0xFF), uint32((bg>>8)&0xFF), uint32((bg>>16)&0xFF)

	for dy := 0; dy < size; dy++ {
		sy := dy * icon.height / size
		for dx := 0; dx < size; dx++ {
			sx := dx * icon.width / size
			off := (sy*icon.width + sx) * 4
			if off+3 >= len(icon.pix) {
				continue
			}

			sr := uint32(icon.pix[off])
			sg := uint32(icon.pix[off+1])
			sb := uint32(icon.pix[off+2])
			sa := uint32(icon.pix[off+3])
			if sa == 0 {
				continue
			}

			or := (sr*sa + bgR*(255-sa)) / 255
			og := (sg*sa + bgG*(255-sa)) / 255
			ob := (sb*sa + bgB*(255-sa)) / 255

			colorRef := uintptr((ob << 16) | (og << 8) | or)
			setPixelV.Call(
				uintptr(hdc),
				uintptr(iconLeft+dx),
				uintptr(iconTop+dy),
				colorRef,
			)
		}
	}

	return true
}

func getTrayMenuIconByID(id uint32) *trayIconPixels {
	palette := currentTrayPalette()
	iconKey := ""

	switch id {
	case ID_SHOW:
		iconKey = "tray_show"
	case ID_HIDE:
		iconKey = "tray_hide"
	case ID_THEME:
		// Use dedicated switch icons: when current is dark, show switch-to-light icon, and vice versa.
		if trayMenuTheme == "dark" {
			iconKey = "tray_switch2light"
		} else {
			iconKey = "tray_switch2dark"
		}
	case ID_QUIT:
		iconKey = "tray_quit"
	default:
		return nil
	}

	cacheKey := iconKey + "_" + palette.iconVariant
	if icon, ok := trayMenuIcons[cacheKey]; ok {
		return icon
	}

	icon := loadTrayIconFile(iconKey + "_" + palette.iconVariant + ".png")
	if icon == nil {
		fallback := "light"
		if palette.iconVariant == "light" {
			fallback = "dark"
		}
		icon = loadTrayIconFile(iconKey + "_" + fallback + ".png")
	}
	trayMenuIcons[cacheKey] = icon
	return icon
}

func loadTrayIconFile(fileName string) *trayIconPixels {
	for _, candidate := range trayIconCandidates(fileName) {
		f, err := os.Open(candidate)
		if err != nil {
			continue
		}

		img, decodeErr := png.Decode(f)
		f.Close()
		if decodeErr != nil {
			continue
		}

		nrgba := toNRGBA(img)
		return &trayIconPixels{
			width:  nrgba.Rect.Dx(),
			height: nrgba.Rect.Dy(),
			pix:    nrgba.Pix,
		}
	}

	return nil
}

func trayIconCandidates(fileName string) []string {
	rel := filepath.Join("assets", "png-tray_icon", fileName)
	candidates := []string{rel}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, rel),
			filepath.Join(exeDir, "..", rel),
			filepath.Join(exeDir, "..", "..", rel),
		)
	}

	return candidates
}

func toNRGBA(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}
