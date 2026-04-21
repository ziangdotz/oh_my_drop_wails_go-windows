<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import * as Runtime from '@wailsjs/runtime'
import DropNull from './components/DropNull.vue'
import DropReady from './components/DropReady.vue'
import DropReadyExpand from './components/DropReadyExpand.vue'

/* 延迟函数 */
const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));

const isMoreDropMenuOpen = ref(false) // 控制 Header 更多按钮左下角菜单的状态
const toggleMenu = () => {
	isMoreDropMenuOpen.value = !isMoreDropMenuOpen.value
}

const getEffectiveTheme = () => {
	const explicit = document.documentElement.getAttribute('data-theme')
	if (explicit === 'dark' || explicit === 'light') {
		return explicit
	}
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

const toggleThemeFromTray = (theme) => {
	const nextTheme = theme === 'dark' || theme === 'light'
		? theme
		: (getEffectiveTheme() === 'dark' ? 'light' : 'dark')
	document.documentElement.setAttribute('data-theme', nextTheme)
}

const hasFileContent = ref(false) // 控制是否有文件内容的状态
const selectedFilePaths = ref([]) // 存储文件路径的列表
const handleFileContentDropped = (filepaths, isInitial = false) => {
	/* 去重并过滤空路径 */
	const newPaths = Array.from(new Set(filepaths)).filter(p => p && p.trim() !== '')
	/* 找出哪些路径是还没有缓存缩略图的 */
	const pathsToFetch = newPaths.filter(p => !thumbnailsMap.value.has(p))
	/* 更新路径列表 */
	if (isInitial) {
		/* 第一次拖入：全量正序排列（自然排序） */
		const combined = [...newPaths]
		combined.sort((a, b) => a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' }))
		selectedFilePaths.value = combined
	} else {
		/* 后续拖入：新文件放到数组首部，旧文件紧随其后（并整体去重） */
		const combined = [...newPaths, ...selectedFilePaths.value]
		selectedFilePaths.value = Array.from(new Set(combined))
	}
	/* 只针对 “没缓存过” 的路径请求 Go 后端，避免内存抖动和重复计算 */
	if (pathsToFetch.length > 0) {
		window.go.main.App.HandleFilePaths_GetThumbnails(pathsToFetch) // 请求 Go 后端生成缩略图
	}
	if (selectedFilePaths.value.length > 0) {
		hasFileContent.value = true
	}
	// console.log('selectedFilePaths:', selectedFilePaths.value)
}

const thumbnailsMap = ref(new Map()) // 使用 Map 存储：path -> {base64, ratio, orientation}
/* 监听后端实时推送的缩略图数据 */
const setupThumbnailsListeners = () => {
	Runtime.EventsOn("thumbnails_ready", (data) => {
		thumbnailsMap.value.set(data.path, data) // 如果缓存中没有，或者数据有更新，则存入 Map
	})
}
/* 计算当前已选中路径对应的缩略图列表（保持顺序） */
const displayImages = computed(() => {
	return selectedFilePaths.value
		.map(path => thumbnailsMap.value.get(path))
		.filter(Boolean) // 只返回已生成缩略图的项
})

/* 小窗只展示前 10 个缩略图，展开页展示全部 */
const displayImagesForReady = computed(() => {
	return displayImages.value.slice(0, 10)
})

const currentPage = ref('ready') // ready（小窗）或 ready_expand（大窗）
const handleWindowExpand = async () => {
	// 1. 获取当前窗口的确切物理坐标
	const pos = await Runtime.WindowGetPosition()
	// 2. 改变窗口大小
	await Runtime.WindowSetSize(600, 400)
	// // 3. 居中窗口（可选，防止窗口只向右下角单向扩张）
	// await Runtime.WindowCenter()
	// 4. 强制将左上角重新定位到刚才记录的坐标，实现“视觉固定”
	await Runtime.WindowSetPosition(pos.x, pos.y)
	currentPage.value = 'ready_expand'
}
const handleWindowRestore = async () => {
	// 1. 获取当前窗口的确切物理坐标
	const pos = await Runtime.WindowGetPosition()
	// 2. 改变窗口大小
	await Runtime.WindowSetSize(210, 230)
	// // 3. 居中窗口（可选，防止窗口只向右下角单向扩张）
	// await Runtime.WindowCenter()
	// 4. 强制将左上角重新定位到刚才记录的坐标，实现“视觉固定”
	await Runtime.WindowSetPosition(pos.x, pos.y)
	currentPage.value = 'ready'
}

/* 处理所有文件内容剪切操作 */
const handleCutAllFileContent = async () => {
	if (selectedFilePaths.value.length > 0) {
		/* 调用 Go 后端暴露的 操作剪切板 方法 */
		if (window.go?.main?.App) {
			window.go.main.App.HandleFilePaths_MarkFileOperate("move").catch(err => {
				console.error("无法执行 HandleFilePaths_MarkFileOperate:", err)
			})
			const res = await window.go.main.App.HandleFilePaths_OpearateWithClipboard(selectedFilePaths.value).catch(err => {
				console.error("无法执行 HandleFilePaths_OpearateWithClipboard:", err)
			})
			if (res == "success") {
				if (currentPage.value === 'ready_expand') {
					await handleWindowRestore();
				}
				/* 剪切成功后清空当前内容 */
				hasFileContent.value = false
				thumbnailsMap.value.clear() // 清空缩略图缓存
				selectedFilePaths.value = []
				/* 通知 Go 后端清空缩略图缓存 */
				if (window.go?.main?.App) {
					window.go.main.App.HandleFilePaths_ClearThumbnailsCache().catch(err => {
						console.error("无法执行 HandleFilePaths_ClearThumbnailsCache:", err)
					})
				}
			} else {
				console.error("剪切操作失败:", res)
			}
		}
	}
}
/* 处理所有文件内容复制操作 */
const handleCopyAllFileContent = async () => {
	if (selectedFilePaths.value.length > 0) {
		/* 调用 Go 后端暴露的 操作剪切板 方法 */
		if (window.go?.main?.App) {
			window.go.main.App.HandleFilePaths_MarkFileOperate("copyMove").catch(err => {
				console.error("无法执行 HandleFilePaths_MarkFileOperate:", err)
			})
			const res = await window.go.main.App.HandleFilePaths_OpearateWithClipboard(selectedFilePaths.value).catch(err => {
				console.error("无法执行 HandleFilePaths_OpearateWithClipboard:", err)
			})
			if (res == "success") {
			} else {
				console.error("复制操作失败:", res)
			}
		}
	}
}

const getPathsFromSelectedIndices = (indices) => {
	return indices.map(index => selectedFilePaths.value[index]).filter(Boolean)
}
/* 实现剪切选中逻辑 */
const handleMoveSelected = async (indices) => {
	const pathsToCut = getPathsFromSelectedIndices(indices)
	if (pathsToCut.length === 0) return
	if (window.go?.main?.App) {
		await window.go.main.App.HandleFilePaths_MarkFileOperate("move")
		const res = await window.go.main.App.HandleFilePaths_OpearateWithClipboard(pathsToCut)
		if (res === "success") {
			handleDeleteSelected(indices)
		}
	}
}
/* 实现复制选中逻辑 */
const handleCopyMoveSelected = async (indices) => {
	const pathsToCopy = getPathsFromSelectedIndices(indices)
	if (pathsToCopy.length === 0) return
	if (window.go?.main?.App) {
		await window.go.main.App.HandleFilePaths_MarkFileOperate("copyMove")
		await window.go.main.App.HandleFilePaths_OpearateWithClipboard(pathsToCopy)
	}
}
/* 实现删除选中逻辑 */
const handleDeleteSelected = (indices) => {
	const pathsToDelete = getPathsFromSelectedIndices(indices)
	selectedFilePaths.value = selectedFilePaths.value.filter((_, index) => !indices.includes(index)) // 更新路径列表：过滤掉在选中索引列表中的项
	pathsToDelete.forEach(path => thumbnailsMap.value.delete(path)) // 从缩略图缓存中移除，释放内存
	/* 如果删完了，返回小窗 */
	if (selectedFilePaths.value.length === 0) {
		handleWindowRestore()
		hasFileContent.value = false
	}
}

const handleMoreDropMenuAction = async (action) => {
	await sleep(250);
	// if (action === 'newWindow') {
	// 	/* 获取当前缩放倍率 */
	// 	const windowDpr = window.devicePixelRatio || 1;
	// 	/* 调用 Go 后端暴露的 NewWindow 方法 */
	// 	if (window.go?.main?.App) {
	// 		window.go.main.App.NewWindow(windowDpr).catch(err => {
	// 			console.error("无法执行 NewWindow:", err)
	// 		})
	// 	}
	// } else if (action === 'cutAll') {
	if (action === 'cutAll') {
		await handleCutAllFileContent()
	} else if (action === 'copyAll') {
		await handleCopyAllFileContent()
	} else if (action === 'clear') {
		if (currentPage.value === 'ready_expand') {
			await handleWindowRestore();
		}
		hasFileContent.value = false
		thumbnailsMap.value.clear() // 清空缩略图缓存
		selectedFilePaths.value = []
		/* 通知 Go 后端清空缩略图缓存 */
		if (window.go?.main?.App) {
			window.go.main.App.HandleFilePaths_ClearThumbnailsCache().catch(err => {
				console.error("无法执行 HandleFilePaths_ClearThumbnailsCache:", err)
			})
		}
	}
	isMoreDropMenuOpen.value = false
}

function hideWindow() {
	/* 通知 Go 后端隐藏应用 */
	if (window.go?.main?.App) {
		window.go.main.App.HideWindow()
	}
}

onMounted(() => {
	// window.addEventListener('contextmenu', (e) => e.preventDefault()); // 禁用整个窗口的右键菜单

	setupThumbnailsListeners() // 启动事件监听
	Runtime.EventsOn("theme_toggle", toggleThemeFromTray)

	/* 监听原生文件拖放事件 */
	Runtime.OnFileDrop((x, y, paths) => {
		// console.log("[Vue + Wails] 捕获到的绝对路径:", paths);
		if (paths && paths.length > 0) {
			handleFileContentDropped(paths, !hasFileContent.value); // 追加模式渲染缩略图
		}
	}, true); // 第二个参数 true 表示只响应带有 --wails-drop-target: drop 属性的元素
})

onUnmounted(() => {
	Runtime.EventsOff("thumbnails_ready"); // 销毁监听
	Runtime.EventsOff("theme_toggle");
	Runtime.OnFileDropOff(); // 销毁监听
})
</script>

<template>
	<div class="container">
		<div class="container_header">
			<button class="container_header_icon_btn"
				@click="hideWindow"
				title="关闭">
				<svg viewBox="0 0 24 24" width="18" height="18" stroke="currentColor" stroke-width="2.5" fill="none">
					<line x1="18" y1="6" x2="6" y2="18"></line>
					<line x1="6" y1="6" x2="18" y2="18"></line>
				</svg>
			</button>

			<div class="container_header_middle">
				<div class="container_header_middle_grabber"> </div>
			</div>

			<div class="container_header_right_more_menu_wrapper">
				<button v-if="hasFileContent" class="container_header_icon_btn"
					@click="toggleMenu"
					title="更多">
					<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
						<circle cx="5" cy="12" r="2" />
						<circle cx="12" cy="12" r="2" />
						<circle cx="19" cy="12" r="2" />
					</svg>
				</button>

				<Transition name="dropdown_menu_zoom">
					<div v-if="isMoreDropMenuOpen" class="dropdown_menu">
						<div class="dropdown_menu_list">
							<!-- <button class="dropdown_menu_list_item"
								@click="handleMoreDropMenuAction('newWindow')">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
									stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
									<path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-14a2 2 0 0 1-2-2v-4"></path>
									<rect x="3" y="3" width="12" height="12" rx="2" ry="2"></rect>
									<line x1="9" y1="9" x2="15" y2="15"></line>
								</svg>
								新建窗口
							</button> -->
							<button class="dropdown_menu_list_item"
								@click="handleMoreDropMenuAction('cutAll')">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
									stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
									<circle cx="6" cy="6" r="3"></circle>
									<circle cx="6" cy="18" r="3"></circle>
									<line x1="20" y1="4" x2="8.12" y2="15.88"></line>
									<line x1="14.47" y1="14.48" x2="20" y2="20"></line>
									<line x1="8.12" y1="8.12" x2="12" y2="12"></line>
								</svg>
								剪切所有
							</button>
							<button class="dropdown_menu_list_item"
								@click="handleMoreDropMenuAction('copyAll')">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
									stroke-width="2">
									<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
									<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
								</svg>
								复制所有
							</button>
							<div class="dropdown_menu_divider"></div>
							<button class="dropdown_menu_list_item danger"
								@click="handleMoreDropMenuAction('clear')">
								<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
									stroke-width="2">
									<polyline points="3 6 5 6 21 6"></polyline>
									<path
										d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2">
									</path>
								</svg>
								清空暂存
							</button>
						</div>
					</div>
				</Transition>
			</div>
		</div>

		<div class="container_content">
			<div v-if="!hasFileContent" class="container_content_inner">
				<DropNull />
			</div>
			<div v-else class="container_content_inner">
				<DropReady v-if="currentPage === 'ready'" key="ready" class="window_absview"
					:images="displayImagesForReady"
					:totalCount="selectedFilePaths.length"
					@windowExpand="handleWindowExpand" />
				<DropReadyExpand v-else-if="currentPage === 'ready_expand'" key="ready_expand" class="window_absview"
					:images="displayImages"
					@windowRestore="handleWindowRestore"
					@moveSelected="handleMoveSelected"
					@copyMoveSelected="handleCopyMoveSelected"
					@deleteSelected="handleDeleteSelected" />
			</div>
		</div>

		<Transition name="overlay_fade">
			<div v-if="isMoreDropMenuOpen" class="global_overlay"
				@click="isMoreDropMenuOpen = false"></div>
		</Transition>
	</div>
</template>

<style>
/* 定义统一的动画参数变量 */
:root {
	--menu-anim-time: 0.2s;
	--menu-bezier: cubic-bezier(0.4, 0, 0.2, 1);
}

* {
	user-select: none; /* 禁止全局文字选中 标准语法 */
	-ms-user-select: none; /* 禁止全局文字选中 IE/Edge */
	-moz-user-select: none; /* 禁止全局文字选中 Firefox */
	-webkit-user-select: none; /* 禁止全局文字选中 Safari, Chrome, WebView2 */
}

body {
	margin: 0;
	background-color: transparent;
	font-family: system-ui, -apple-system, sans-serif;
}

.container {
	width: 100%;
	height: 100%;
	padding: 7.5px 15px 15px 15px;
	box-sizing: border-box;
	background-color: var(--bg-surface-1);
	color: var(--text-main);
	display: flex;
	flex-direction: column;
	overflow: hidden;
}

/******** Header ********/
.container_header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 0px 0px 10px 0px;
	position: relative;
}

/**** Header 两侧 ****/
/** Header 两侧的按钮 **/
.container_header_icon_btn {
	--wails-draggable: no-drag; /* 禁止按钮区域的拖动，否则点击无效 */
	width: 24px;
	height: 24px;
	border-radius: 50%;
	border: none;
	background-color: var(--bg-surface-3);
	color: var(--text-faint);
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: background-color 0.2s;
}
.container_header_icon_btn:hover {
	background-color: var(--bg-surface-2);
	color: var(--text-main);
}

.container_header_right_more_menu_wrapper {
	position: relative;
	display: flex;
	align-items: center;
}

.dropdown_menu {
	position: absolute;
	top: 29px; /* 按钮高度 24px + 间距 5px */
	right: 0; /* 右边对齐 */
	width: max-content; /* 宽度根据内容最大长度自动调整 */
	min-width: 110px;
	height: max-content; /* 高度根据内容最大长度自动调整 */
	max-height: 170px;
	overflow-y: auto; /* 支持滚动 */
	scrollbar-width: none; /* 隐藏滚动条 */
	-ms-overflow-style: none; /* 隐藏滚动条 IE 10+ */
	background-color: var(--bg-elevated);
	backdrop-filter: blur(12px);
	-webkit-backdrop-filter: blur(12px); /* 兼容性其他内核 */
	background-clip: padding-box; /* 确保背景模糊只作用于内容区域 */
	border: 1px solid var(--border-soft);
	border-radius: 10px;
	padding: 0px;
	/* box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1); */
	z-index: 2010;
}

.dropdown_menu_list_item {
	width: 100%;
	padding: 8px 10px;
	background: transparent;
	border: none;
	border-radius: 10px;
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 10px;
	color: var(--text-main);
	font-size: 14px;
	font-weight: 400;
	text-align: center;
	line-height: 1.2;
	transition: background 0.2s;
	cursor: pointer;
}
.dropdown_menu_list_item:hover {
	background-color: var(--bg-surface-3);
	color: var(--text-main);
	font-weight: 600;
}
.dropdown_menu_list_item.danger {
	color: var(--danger);
}
.dropdown_menu_list_item.danger:hover {
	background-color: var(--danger-soft);
	font-weight: 600;
}

.dropdown_menu_list_item svg {
	display: block;
	flex-shrink: 0;
}

.dropdown_menu_divider {
	margin: 4px 5px;
	height: 1px;
	background-color: var(--divider);
}

/* 菜单动画 */
.dropdown_menu_zoom-enter-active,
.dropdown_menu_zoom-leave-active {
	transition:
		opacity var(--menu-anim-time) var(--menu-bezier),
		transform var(--menu-anim-time) var(--menu-bezier);
}

.dropdown_menu_zoom-enter-from,
.dropdown_menu_zoom-leave-to {
	opacity: 0;
	transform: translateY(-5px) scale(0.9); /* 略微缩小并上移 */
	transform-origin: top right; /* 动画从右上角开始 */
}

/**** Header 中间 ****/
.container_header_middle {
	--wails-draggable: drag; /* 可以拖动 */
	width: 40px;
	height: 24px;
	position: absolute;
	left: 50%;
	transform: translateX(-50%);
	display: flex;
	justify-content: center;
	cursor: grab;
}

/** Header 中间的小提手 **/
.container_header_middle_grabber {
	width: 40px;
	height: 4px;
	margin: 0px;
	background-color: var(--bg-surface-3);
	border-radius: 4px;
}

.container_content {
	--wails-drop-target: drop; /* 告诉 Wails 内核这是一个原生掉落目标 */
	position: relative;
	width: 100%;
	flex: 1; /* 占据 Header 之外的所有空间 */
	display: flex;
	flex-direction: column;
	min-height: 0;
}

.container_content_inner {
	position: relative;
	width: 100%;
	flex: 1; /* 占据 Header 之外的所有空间 */
	display: flex;
	flex-direction: column;
	min-height: 0;
}

/* 绝对定位并固定左上角 */
.window_absview {
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	transform-origin: left top;
	backface-visibility: hidden;
}

/******** 全局遮罩 ********/
.global_overlay {
	position: absolute;
	inset: 0;
	z-index: 2005;
	background-color: var(--overlay-soft);
	backdrop-filter: blur(0.9px) saturate(180%) brightness(105%);
	-webkit-backdrop-filter: blur(0.9px) saturate(180%) brightness(105%); /* 兼容性其他内核 */
	background: transparent;
	pointer-events: auto; /* 确保拦截点击 */
}

/* 全局遮罩动画 */
.overlay-fade-enter-active,
.overlay-fade-leave-active {
	transition: opacity var(--menu-anim-time) var(--menu-bezier);
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
	opacity: 0;
}
</style>
