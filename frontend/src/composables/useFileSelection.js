export function useFileSelection(selectedFilePaths, hasFileContent, thumbnailsMap, handleWindowRestore, clearThumbnailsCache) {
	const getPathsFromSelectedIndices = (indices) => {
		return indices.map(index => selectedFilePaths.value[index]).filter(Boolean)
	}

	const handleDeleteSelected = (indices) => {
		const pathsToDelete = getPathsFromSelectedIndices(indices)
		selectedFilePaths.value = selectedFilePaths.value.filter((_, index) => !indices.includes(index))
		pathsToDelete.forEach(path => thumbnailsMap.value.delete(path))
		if (selectedFilePaths.value.length === 0) {
			handleWindowRestore()
			hasFileContent.value = false
		}
	}

	const handleMoveSelected = async (indices) => {
		const pathsToCut = getPathsFromSelectedIndices(indices)
		if (pathsToCut.length === 0) return
		if (window.go?.main?.App) {
			await window.go.main.App.HandleFilePaths_MarkFileOperate("move")
			const res = await window.go.main.App.HandleFilePaths_OperateWithClipboard(pathsToCut)
			if (res === "success") {
				handleDeleteSelected(indices)
			}
		}
	}

	const handleCopyMoveSelected = async (indices) => {
		const pathsToCopy = getPathsFromSelectedIndices(indices)
		if (pathsToCopy.length === 0) return
		if (window.go?.main?.App) {
			await window.go.main.App.HandleFilePaths_MarkFileOperate("copyMove")
			await window.go.main.App.HandleFilePaths_OperateWithClipboard(pathsToCopy)
		}
	}

	const handleCutAllFileContent = async () => {
		if (selectedFilePaths.value.length > 0) {
			if (window.go?.main?.App) {
				window.go.main.App.HandleFilePaths_MarkFileOperate("move").catch(err => {
					console.error("无法执行 HandleFilePaths_MarkFileOperate:", err)
				})
				const res = await window.go.main.App.HandleFilePaths_OperateWithClipboard(selectedFilePaths.value).catch(err => {
					console.error("无法执行 HandleFilePaths_OperateWithClipboard:", err)
				})
				if (res === "success") {
					await handleWindowRestore()
					hasFileContent.value = false
					thumbnailsMap.value.clear()
					selectedFilePaths.value = []
					clearThumbnailsCache()
				} else {
					console.error("剪切操作失败:", res)
				}
			}
		}
	}

	const handleCopyAllFileContent = async () => {
		if (selectedFilePaths.value.length > 0) {
			if (window.go?.main?.App) {
				window.go.main.App.HandleFilePaths_MarkFileOperate("copyMove").catch(err => {
					console.error("无法执行 HandleFilePaths_MarkFileOperate:", err)
				})
				const res = await window.go.main.App.HandleFilePaths_OperateWithClipboard(selectedFilePaths.value).catch(err => {
					console.error("无法执行 HandleFilePaths_OperateWithClipboard:", err)
				})
				if (res !== "success") {
					console.error("复制操作失败:", res)
				}
			}
		}
	}

	const handleClearAll = async (currentPage, handleWindowRestoreFn) => {
		if (currentPage.value === 'ready_expand') {
			await handleWindowRestoreFn()
		}
		hasFileContent.value = false
		thumbnailsMap.value.clear()
		selectedFilePaths.value = []
		clearThumbnailsCache()
	}

	return {
		getPathsFromSelectedIndices,
		handleDeleteSelected,
		handleMoveSelected,
		handleCopyMoveSelected,
		handleCutAllFileContent,
		handleCopyAllFileContent,
		handleClearAll,
	}
}
