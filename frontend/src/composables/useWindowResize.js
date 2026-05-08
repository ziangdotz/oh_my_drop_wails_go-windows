import { ref } from 'vue'
import * as Runtime from '@wailsjs/runtime'

export function useWindowResize() {
	const currentPage = ref('ready')

	const handleWindowExpand = async () => {
		const pos = await Runtime.WindowGetPosition()
		await Runtime.WindowSetSize(600, 400)
		await Runtime.WindowSetPosition(pos.x, pos.y)
		currentPage.value = 'ready_expand'
	}

	const handleWindowRestore = async () => {
		const pos = await Runtime.WindowGetPosition()
		await Runtime.WindowSetSize(210, 230)
		await Runtime.WindowSetPosition(pos.x, pos.y)
		currentPage.value = 'ready'
	}

	return {
		currentPage,
		handleWindowExpand,
		handleWindowRestore,
	}
}
