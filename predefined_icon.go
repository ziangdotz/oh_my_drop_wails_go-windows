package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"image"
	"image/png"

	"github.com/disintegration/imaging"
)

// 预置文件图标以 PNG 资源形式嵌入，避免在源码中内联大段 Base64 字符串。
//
//go:embed assets/png-predefined_icon/*.png
var predefinedIconFS embed.FS

// IconData 存储预置图标的 Base64 数据与宽高比
type IconData struct {
	Base64 string
	Ratio  float64
}

// predefinedIcons 在初始化时从嵌入资源解码生成，键为文件类型常量。
var predefinedIcons = buildPredefinedIcons()

// predefinedIconMaxEdge 为预置图标的长边上限。预置图标会在每个同类文件的
// 缩略图中被重复下发，原始 512px PNG 单份 Base64 达 16~34KB，造成大量
// 冗余传输；降采样到该尺寸后体积减小数倍，而简单图标视觉上几乎无损。
const predefinedIconMaxEdge = 256

// buildPredefinedIcons 读取嵌入的 PNG，降采样后生成 data URI 与宽高比。
func buildPredefinedIcons() map[string]IconData {
	files := map[string]string{
		TypeFolder: "folder.png",
		TypeZip:    "archive.png",
		TypeExe:    "executable.png",
		TypeFile:   "file.png",
	}

	icons := make(map[string]IconData, len(files))
	for fileType, name := range files {
		data, err := predefinedIconFS.ReadFile("assets/png-predefined_icon/" + name)
		if err != nil {
			continue
		}

		// 默认直接使用原始字节；仅在成功解码且尺寸超限时降采样重编码。
		encoded := data
		ratio := 1.0
		if img, _, derr := image.Decode(bytes.NewReader(data)); derr == nil {
			b := img.Bounds()
			w, h := b.Dx(), b.Dy()
			if h > 0 {
				ratio = float64(w) / float64(h)
			}
			if w > predefinedIconMaxEdge || h > predefinedIconMaxEdge {
				if w >= h {
					img = imaging.Resize(img, predefinedIconMaxEdge, 0, imaging.CatmullRom)
				} else {
					img = imaging.Resize(img, 0, predefinedIconMaxEdge, imaging.CatmullRom)
				}
				var buf bytes.Buffer
				if eerr := png.Encode(&buf, img); eerr == nil {
					encoded = buf.Bytes()
				}
			}
		}

		icons[fileType] = IconData{
			Base64: "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded),
			Ratio:  ratio,
		}
	}
	return icons
}

// HandleFilePaths_GetPredefinedIcons 返回各文件类型对应的预置图标 data URI（键为
// TypeFolder/TypeZip/TypeExe/TypeFile）。前端在启动时拉取并缓存一次，之后
// 非图片文件的缩略图仅按 iconKey 引用，无需重复下发 base64。
func (a *App) HandleFilePaths_GetPredefinedIcons() map[string]string {
	result := make(map[string]string, len(predefinedIcons))
	for fileType, icon := range predefinedIcons {
		result[fileType] = icon.Base64
	}
	return result
}
