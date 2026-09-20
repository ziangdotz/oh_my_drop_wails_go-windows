//go:build windows

package main

import (
	"golang.org/x/sys/windows"
)

// 本文件集中声明托盘/菜单绘制所需的 Win32 常量、结构体与 DLL 过程句柄，
// 与具体的业务逻辑（消息循环、绘制、主题、图标）分离，便于查阅与维护。

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
	SRCCOPY             = 0x00CC0020
	DIB_RGB_COLORS      = 0
	BI_RGB              = 0
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

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type MEASUREITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemWidth  uint32
	ItemHeight uint32
	ItemData   uintptr
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
	getDpiForWindow  = trayUser32.NewProc("GetDpiForWindow")

	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	createDIBSection   = gdi32.NewProc("CreateDIBSection")
	deleteDC           = gdi32.NewProc("DeleteDC")
	bitBlt             = gdi32.NewProc("BitBlt")
)
