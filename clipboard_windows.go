//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	setClipboardData = user32.NewProc("SetClipboardData")
	registerFormat   = user32.NewProc("RegisterClipboardFormatW")

	kernel32     = windows.NewLazySystemDLL("kernel32.dll")
	globalAlloc  = kernel32.NewProc("GlobalAlloc")
	globalFree   = kernel32.NewProc("GlobalFree")
	globalLock   = kernel32.NewProc("GlobalLock")
	globalUnlock = kernel32.NewProc("GlobalUnlock")

	keybd_event = user32.NewProc("keybd_event")
)

const (
	CF_HDROP        = 15
	GMEM_MOVEABLE   = 0x0002
	GMEM_ZEROINIT   = 0x0040
	DROPEFFECT_COPY = 1
	DROPEFFECT_MOVE = 2

	VK_CONTROL      = 0x11
	VK_V            = 0x56
	KEYEVENTF_KEYUP = 0x0002
)

type dropFiles struct {
	pFiles uint32
	pt     [8]byte
	fNC    int32
	fWide  int32
}

func (a *App) writeFilesToClipboard(paths []string, isMove bool) error {
	var buffer []uint16
	for _, p := range paths {
		absPath, _ := filepath.Abs(filepath.FromSlash(p)) // 强制转换为 Windows 原生路径格式
		u16, _ := windows.UTF16FromString(absPath)
		buffer = append(buffer, u16...)
	}
	buffer = append(buffer, 0) // 双 NULL 结尾

	dfSize := uint32(unsafe.Sizeof(dropFiles{}))
	totalSize := dfSize + uint32(len(buffer)*2)

	hMem, _, _ := globalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, uintptr(totalSize))
	if hMem == 0 {
		return fmt.Errorf("内存分配失败")
	}

	ptr, _, _ := globalLock.Call(hMem)
	df := (*dropFiles)(unsafe.Pointer(ptr))
	df.pFiles = dfSize
	df.fWide = 1

	dataPtr := unsafe.Pointer(uintptr(ptr) + uintptr(dfSize)) // 使用更安全的拷贝方式
	for i, v := range buffer {
		*(*uint16)(unsafe.Pointer(uintptr(dataPtr) + uintptr(i*2))) = v
	}
	globalUnlock.Call(hMem)

	ret, _, _ := openClipboard.Call(0)
	if ret == 0 {
		globalFree.Call(hMem)
		return fmt.Errorf("OpenClipboard 失败")
	}
	defer closeClipboard.Call()
	emptyClipboard.Call()
	if h, _, _ := setClipboardData.Call(CF_HDROP, hMem); h == 0 {
		globalFree.Call(hMem)
		return fmt.Errorf("SetClipboardData 失败")
	}

	formatName, _ := windows.UTF16PtrFromString("Preferred DropEffect")
	cfDropEffect, _, _ := registerFormat.Call(uintptr(unsafe.Pointer(formatName)))
	hEffect, _, _ := globalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, 4)
	ePtr, _, _ := globalLock.Call(hEffect)
	if isMove {
		*(*uint32)(unsafe.Pointer(ePtr)) = DROPEFFECT_MOVE
	} else {
		*(*uint32)(unsafe.Pointer(ePtr)) = DROPEFFECT_COPY
	}
	globalUnlock.Call(hEffect)
	setClipboardData.Call(cfDropEffect, hEffect)
	return nil
}
