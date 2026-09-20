//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// 本文件负责托盘菜单项的自绘（owner-draw）：背景、分隔线、版本号、
// 图标与文字的绘制，以及基于 DIB 的 PNG 图标批量 blit。

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

	// 创建内存 DC 与 32bpp 自顶向下 DIB，一次性批量写入像素后再 BitBlt，
	// 避免逐像素调用 SetPixelV（size*size 次系统调用）带来的开销。
	memDC, _, _ := createCompatibleDC.Call(uintptr(hdc))
	if memDC == 0 {
		return false
	}
	defer deleteDC.Call(memDC)

	var bmi BITMAPINFO
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(BITMAPINFOHEADER{}))
	bmi.BmiHeader.BiWidth = int32(size)
	bmi.BmiHeader.BiHeight = -int32(size) // 负值表示自顶向下
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = BI_RGB

	var bits unsafe.Pointer
	hbmp, _, _ := createDIBSection.Call(
		memDC,
		uintptr(unsafe.Pointer(&bmi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&bits)),
		0, 0,
	)
	if hbmp == 0 || bits == nil {
		return false
	}
	defer deleteObject.Call(hbmp)

	// DIB 为 BGRA 字节序；将图标按 alpha 混合到菜单背景色（sa==0 时公式自然得到背景色）。
	buf := unsafe.Slice((*uint8)(bits), size*size*4)
	for dy := 0; dy < size; dy++ {
		sy := dy * icon.height / size
		for dx := 0; dx < size; dx++ {
			sx := dx * icon.width / size
			off := (sy*icon.width + sx) * 4
			o := (dy*size + dx) * 4
			if off+3 >= len(icon.pix) {
				buf[o], buf[o+1], buf[o+2], buf[o+3] = uint8(bgB), uint8(bgG), uint8(bgR), 255
				continue
			}

			sr := uint32(icon.pix[off])
			sg := uint32(icon.pix[off+1])
			sb := uint32(icon.pix[off+2])
			sa := uint32(icon.pix[off+3])

			or := (sr*sa + bgR*(255-sa)) / 255
			og := (sg*sa + bgG*(255-sa)) / 255
			ob := (sb*sa + bgB*(255-sa)) / 255

			buf[o], buf[o+1], buf[o+2], buf[o+3] = uint8(ob), uint8(og), uint8(or), 255
		}
	}

	oldBmp, _, _ := selectObject.Call(memDC, hbmp)
	bitBlt.Call(
		uintptr(hdc),
		uintptr(iconLeft), uintptr(iconTop),
		uintptr(size), uintptr(size),
		memDC, 0, 0,
		SRCCOPY,
	)
	if oldBmp != 0 {
		selectObject.Call(memDC, oldBmp)
	}

	return true
}
