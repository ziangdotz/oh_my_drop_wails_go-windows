import { ref } from 'vue'
import * as Runtime from '@wailsjs/runtime'

// 窗口尺寸集中定义，避免魔数分散在多个组件中。
// compact 需与 main.go 中的初始 Width/Height 保持一致。
export const WINDOW_SIZE = {
	compact: { width: 210, height: 230 },
	expanded: { width: 600, height: 400 },
}

export function useWindowResize() {
	const currentPage = ref('ready')

	const handleWindowExpand = async () => {
		await Runtime.WindowSetSize(WINDOW_SIZE.expanded.width, WINDOW_SIZE.expanded.height)
		await Runtime.WindowCenter()
		currentPage.value = 'ready_expand'
	}

	const handleWindowRestore = async () => {
		const pos = await Runtime.WindowGetPosition()
		await Runtime.WindowSetSize(WINDOW_SIZE.compact.width, WINDOW_SIZE.compact.height)
		await Runtime.WindowSetPosition(pos.x, pos.y)
		currentPage.value = 'ready'
	}

	return {
		currentPage,
		handleWindowExpand,
		handleWindowRestore,
	}
}
