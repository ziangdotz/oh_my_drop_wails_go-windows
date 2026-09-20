//go:build windows

package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	"image/png"
)

// 本文件负责托盘菜单图标的加载与缓存：以 go:embed 打包 PNG 资源，
// 解码为 NRGBA 像素后按“图标名_主题变体”缓存，供绘制逻辑使用。

// 托盘菜单图标以 go:embed 方式打包进二进制，避免运行时依赖磁盘上的相对路径。
//
//go:embed assets/png-tray_icon/*.png
var trayIconFS embed.FS

var trayMenuIcons = map[string]*trayIconPixels{}

type trayIconPixels struct {
	width  int
	height int
	pix    []uint8 // NRGBA pixels
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
	data, err := trayIconFS.ReadFile("assets/png-tray_icon/" + fileName)
	if err != nil {
		return nil
	}

	img, decodeErr := png.Decode(bytes.NewReader(data))
	if decodeErr != nil {
		return nil
	}

	nrgba := toNRGBA(img)
	return &trayIconPixels{
		width:  nrgba.Rect.Dx(),
		height: nrgba.Rect.Dy(),
		pix:    nrgba.Pix,
	}
}

func toNRGBA(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}
