import { ref, computed } from 'vue'
import * as Runtime from '@wailsjs/runtime'

export function useThumbnails(selectedFilePaths, hasFileContent) {
	const thumbnailsMap = ref(new Map())
	// 预置图标（folder/archive/executable/file）在启动时拉取一次并本地缓存，
	// 之后非图片文件的缩略图仅带 iconKey，由此映射补齐 base64，避免重复传输。
	const predefinedIcons = ref({})

	const loadPredefinedIcons = async () => {
		try {
			if (window.go?.main?.App?.HandleFilePaths_GetPredefinedIcons) {
				const icons = await window.go.main.App.HandleFilePaths_GetPredefinedIcons()
				predefinedIcons.value = icons || {}
				// 兑底：回填图标就绪前已收到、但缺失 base64 的条目
				thumbnailsMap.value.forEach((item) => {
					if (!item.base64 && item.iconKey && predefinedIcons.value[item.iconKey]) {
						item.base64 = predefinedIcons.value[item.iconKey]
					}
				})
			}
		} catch (err) {
			console.error("无法加载预置图标:", err)
		}
	}

	const setupThumbnailsListeners = () => {
		Runtime.EventsOn("thumbnails_ready", (data) => {
			if (!data.base64 && data.iconKey && predefinedIcons.value[data.iconKey]) {
				data.base64 = predefinedIcons.value[data.iconKey]
			}
			thumbnailsMap.value.set(data.path, data)
		})
	}

	const displayImages = computed(() => {
		return selectedFilePaths.value
			.map(path => thumbnailsMap.value.get(path))
			.filter(Boolean)
	})

	const displayImagesForReady = computed(() => {
		return displayImages.value.slice(0, 10)
	})

	const handleFileContentDropped = (filepaths, isInitial = false) => {
		const newPaths = Array.from(new Set(filepaths)).filter(p => p && p.trim() !== '')
		const pathsToFetch = newPaths.filter(p => !thumbnailsMap.value.has(p))

		if (isInitial) {
			const combined = [...newPaths]
			combined.sort((a, b) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' }))
			selectedFilePaths.value = combined
		} else {
			const combined = [...newPaths, ...selectedFilePaths.value]
			selectedFilePaths.value = Array.from(new Set(combined))
		}

		if (pathsToFetch.length > 0) {
			window.go.main.App.HandleFilePaths_GetThumbnails(pathsToFetch)
		}
		if (selectedFilePaths.value.length > 0) {
			hasFileContent.value = true
		}
	}

	const clearThumbnailsCache = () => {
		if (window.go?.main?.App) {
			window.go.main.App.HandleFilePaths_ClearThumbnailsCache().catch(err => {
				console.error("无法执行 HandleFilePaths_ClearThumbnailsCache:", err)
			})
		}
	}

	return {
		thumbnailsMap,
		loadPredefinedIcons,
		setupThumbnailsListeners,
		displayImages,
		displayImagesForReady,
		handleFileContentDropped,
		clearThumbnailsCache,
	}
}
