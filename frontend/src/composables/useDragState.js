import { ref } from 'vue'

// 统一的拖拽视觉状态处理，供各拖放区域组件复用，避免重复实现 dragenter/leave/over/drop。
export function useDragState() {
	const isDragging = ref(false)

	const handleDragEnter = (e) => {
		e.preventDefault()
		isDragging.value = true
	}
	const handleDragLeave = (e) => {
		if (e.currentTarget.contains(e.relatedTarget)) return
		isDragging.value = false
	}
	const handleDragOver = (e) => {
		e.preventDefault()
		isDragging.value = true
	}
	const handleDrop = (e) => {
		e.preventDefault()
		isDragging.value = false
	}

	return {
		isDragging,
		handleDragEnter,
		handleDragLeave,
		handleDragOver,
		handleDrop,
	}
}
