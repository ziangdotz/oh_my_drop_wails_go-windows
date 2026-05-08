import { ref, computed } from 'vue'
import * as Runtime from '@wailsjs/runtime'

export function useThumbnails(selectedFilePaths, hasFileContent) {
	const thumbnailsMap = ref(new Map())

	const setupThumbnailsListeners = () => {
		Runtime.EventsOn("thumbnails_ready", (data) => {
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
		setupThumbnailsListeners,
		displayImages,
		displayImagesForReady,
		handleFileContentDropped,
		clearThumbnailsCache,
	}
}
