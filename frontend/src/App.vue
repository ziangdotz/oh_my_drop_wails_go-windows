<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import * as Runtime from '@wailsjs/runtime'
import DropNull from './components/DropNull.vue'
import DropReady from './components/DropReady.vue'
import DropReadyExpand from './components/DropReadyExpand.vue'
import { useThumbnails } from './composables/useThumbnails.js'
import { useFileSelection } from './composables/useFileSelection.js'
import { useWindowResize } from './composables/useWindowResize.js'

const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms))

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

const { currentPage, handleWindowExpand, handleWindowRestore } = useWindowResize()

const selectedFilePaths = ref([])
const hasFileContent = ref(false)

const { thumbnailsMap, setupThumbnailsListeners, displayImages, displayImagesForReady, handleFileContentDropped, clearThumbnailsCache } = useThumbnails(selectedFilePaths, hasFileContent)

const {
	handleDeleteSelected,
	handleMoveSelected,
	handleCopyMoveSelected,
	handleCutAllFileContent,
	handleCopyAllFileContent,
	handleClearAll,
} = useFileSelection(selectedFilePaths, hasFileContent, thumbnailsMap, handleWindowRestore, clearThumbnailsCache)

const isMoreDropMenuOpen = ref(false)
const toggleMenu = () => {
	isMoreDropMenuOpen.value = !isMoreDropMenuOpen.value
}

const handleMoreDropMenuAction = async (action) => {
	await sleep(250)
	if (action === 'cutAll') {
		await handleCutAllFileContent()
	} else if (action === 'copyAll') {
		await handleCopyAllFileContent()
	} else if (action === 'clear') {
		await handleClearAll(currentPage, handleWindowRestore)
	}
	isMoreDropMenuOpen.value = false
}

function hideWindow() {
	if (window.go?.main?.App) {
		window.go.main.App.HideWindow()
	}
}

onMounted(() => {
	document.addEventListener('contextmenu', (e) => e.preventDefault())
	setupThumbnailsListeners()
	Runtime.EventsOn("theme_toggle", toggleThemeFromTray)

	Runtime.OnFileDrop((x, y, paths) => {
		if (paths && paths.length > 0) {
			handleFileContentDropped(paths, !hasFileContent.value)
		}
	}, true)
})

onUnmounted(() => {
	Runtime.EventsOff("thumbnails_ready")
	Runtime.EventsOff("theme_toggle")
	Runtime.OnFileDropOff()
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
:root {
	--menu-anim-time: 0.2s;
	--menu-bezier: cubic-bezier(0.4, 0, 0.2, 1);
}

* {
	user-select: none;
	-ms-user-select: none;
	-moz-user-select: none;
	-webkit-user-select: none;
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

.container_header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 0px 0px 10px 0px;
	position: relative;
}

.container_header_icon_btn {
	--wails-draggable: no-drag;
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
	top: 29px;
	right: 0;
	width: max-content;
	min-width: 110px;
	height: max-content;
	max-height: 170px;
	overflow-y: auto;
	scrollbar-width: none;
	-ms-overflow-style: none;
	background-color: var(--bg-elevated);
	backdrop-filter: blur(12px);
	-webkit-backdrop-filter: blur(12px);
	background-clip: padding-box;
	border: 1px solid var(--border-soft);
	border-radius: 10px;
	padding: 0px;
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

.dropdown_menu_zoom-enter-active,
.dropdown_menu_zoom-leave-active {
	transition:
		opacity var(--menu-anim-time) var(--menu-bezier),
		transform var(--menu-anim-time) var(--menu-bezier);
}

.dropdown_menu_zoom-enter-from,
.dropdown_menu_zoom-leave-to {
	opacity: 0;
	transform: translateY(-5px) scale(0.9);
	transform-origin: top right;
}

.container_header_middle {
	--wails-draggable: drag;
	width: 40px;
	height: 24px;
	position: absolute;
	left: 50%;
	transform: translateX(-50%);
	display: flex;
	justify-content: center;
	cursor: grab;
}

.container_header_middle_grabber {
	width: 40px;
	height: 4px;
	margin: 0px;
	background-color: var(--bg-surface-3);
	border-radius: 4px;
}

.container_content {
	--wails-drop-target: drop;
	position: relative;
	width: 100%;
	flex: 1;
	display: flex;
	flex-direction: column;
	min-height: 0;
}

.container_content_inner {
	position: relative;
	width: 100%;
	flex: 1;
	display: flex;
	flex-direction: column;
	min-height: 0;
}

.window_absview {
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	transform-origin: left top;
	backface-visibility: hidden;
}

.global_overlay {
	position: absolute;
	inset: 0;
	z-index: 2005;
	background-color: var(--overlay-soft);
	backdrop-filter: blur(0.9px) saturate(180%) brightness(105%);
	-webkit-backdrop-filter: blur(0.9px) saturate(180%) brightness(105%);
	background: transparent;
	pointer-events: auto;
}

.overlay-fade-enter-active,
.overlay-fade-leave-active {
	transition: opacity var(--menu-anim-time) var(--menu-bezier);
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
	opacity: 0;
}
</style>
