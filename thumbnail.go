package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"sync"

	"github.com/disintegration/imaging"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// 本文件负责文件缩略图的生成与缓存接入：图片经 imaging 缩放并编码为 JPEG，
// 其他类型回退到预置图标；缓存实现见 thumbnail_cache.go（有上限 LRU）。

type Thumbnail struct {
	Path        string  `json:"path"`
	Base64      string  `json:"base64"`
	IconKey     string  `json:"iconKey"`     // 非图片文件的预置图标类型；图片为空
	Ratio       float64 `json:"ratio"`       // 宽度 / 高度
	Orientation string  `json:"orientation"` // landscape 或 portrait
}

// 缩略图清晰度参数：按长边等比缩放，目标覆盖高 DPI（约 2x）下的显示需求；
// Quality 与插值算法在「清晰度」与「数据量/编码耗时」之间取平衡。
// 注：单张耗时瓶颈在原图解码，提高输出分辨率对总速度影响很小。
const (
	thumbMaxEdge     = 320 // 缩略图长边最大像素
	thumbJPEGQuality = 80  // JPEG 编码质量（0-100）
)

var thumbnailsCache = newThumbnailLRU(thumbnailCacheCapacity)

// 生成缩略图
func (a *App) HandleFilePaths_GetThumbnails(paths []string) {
	sem := make(chan struct{}, 5) // 限制最大并发处理数为 5，防止 CPU 和内存瞬间过载
	var wg sync.WaitGroup

	for _, path := range paths {
		/* 检查缓存 */
		if val, ok := thumbnailsCache.Load(path); ok {
			wailsRuntime.EventsEmit(a.ctx, "thumbnails_ready", val)
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		/* 开启协程并行处理每一张图片 */
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量
			fileType := getFileType(p)
			var data Thumbnail
			if fileType == TypeImage {
				// 1. 打开图片并自动处理 EXIF 旋转方向
				src, err := imaging.Open(p, imaging.AutoOrientation(true))
				if err != nil {
					return
				}
				bounds := src.Bounds()
				sw, sh := bounds.Dx(), bounds.Dy()
				// 2. 按长边等比缩放：仅当原图大于目标时才缩小，避免放大小图反而模糊；
				//    CatmullRom（双三次）比 Linear 更锐利，其开销相对解码可忽略。
				thumb := src
				if sw > thumbMaxEdge || sh > thumbMaxEdge {
					if sw >= sh {
						thumb = imaging.Resize(src, thumbMaxEdge, 0, imaging.CatmullRom)
					} else {
						thumb = imaging.Resize(src, 0, thumbMaxEdge, imaging.CatmullRom)
					}
				}
				// 3. 编码为 JPEG Base64
				buf := new(bytes.Buffer)
				err = jpeg.Encode(buf, thumb, &jpeg.Options{Quality: thumbJPEGQuality})
				if err != nil {
					return
				}
				// 4. 使用原始尺寸计算精确比例
				w, h := float64(sw), float64(sh)
				ratio := w / h
				// 5. 判定横竖屏状态
				orientation := "portrait"
				if ratio > 1 {
					orientation = "landscape"
				}

				data = Thumbnail{
					Path:        p,
					Base64:      fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(buf.Bytes())),
					Ratio:       ratio,
					Orientation: orientation,
				}
			} else {
				icon := predefinedIcons[fileType]
				orientation := "portrait"
				if icon.Ratio > 1 {
					orientation = "landscape"
				}
				// 非图片文件不下发 base64，仅传 iconKey，
				// 由前端本地缓存的预置图标映射，避免同类文件重复传输。
				data = Thumbnail{
					Path:        p,
					IconKey:     fileType,
					Ratio:       icon.Ratio,
					Orientation: orientation,
				}
			}

			thumbnailsCache.Store(p, data) // 存入缓存
			wailsRuntime.EventsEmit(a.ctx, "thumbnails_ready", data)
		}(path)
	}
	wg.Wait() // 等待所有任务完成
}

// 清除缩略图缓存
func (a *App) HandleFilePaths_ClearThumbnailsCache() {
	thumbnailsCache.Clear()
}
