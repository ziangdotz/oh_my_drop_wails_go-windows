//go:build windows

package main

import (
	"math"
)

// 本文件负责托盘菜单的主题（配色）、尺寸度量与 DPI 缩放，
// 与消息循环、绘制逻辑分离，便于集中调整视觉风格。

var trayMenuTheme = "dark"

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
