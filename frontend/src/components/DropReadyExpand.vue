<script setup>
import { ref, computed } from 'vue'

const emit = defineEmits(['windowRestore', 'moveSelected', 'copyMoveSelected', 'deleteSelected'])
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

const props = defineProps({
	images: {
		type: Array,
		default: () => []
	},
})

const handleWindowRestore = async () => {
	emit('windowRestore')
}

const filesStackViewMode = ref('grid')
const fileItemSelectedPaths = ref(new Set())
const fileItemToggleSelect = (path) => {
	const newSet = new Set(fileItemSelectedPaths.value)
	if (newSet.has(path)) {
		newSet.delete(path)
	} else {
		newSet.add(path)
	}
	fileItemSelectedPaths.value = newSet
}
const getFileNameFromImg = (path) => {
	if (!path) return 'Unknown'
	return path.split(/[\\/]/).pop()
}

const fileNameSortType = ref('nameAsc')
const fileNameToggleSortType = () => {
	fileNameSortType.value = fileNameSortType.value === 'nameAsc' ? 'nameDesc' : 'nameAsc'
}
const fileNameSorted = computed(() => {
	let list = [...props.images]
	return list.sort((a, b) => {
		const nameA = getFileNameFromImg(a.path).toLowerCase()
		const nameB = getFileNameFromImg(b.path).toLowerCase()
		if (fileNameSortType.value === 'nameAsc') {
			return nameA.localeCompare(nameB, 'zh-CN', { numeric: true })
		} else {
			return nameB.localeCompare(nameA, 'zh-CN', { numeric: true })
		}
	})
})

const handleAction = (actionType) => {
	if (fileItemSelectedPaths.value.size === 0) {
		return
	}
	const fileItemSelectedIndicesArray = props.images.map((img, index) => fileItemSelectedPaths.value.has(img.path) ? index : -1).filter(idx => idx !== -1)
	emit(actionType, fileItemSelectedIndicesArray)
	fileItemSelectedPaths.value = new Set()
}
</script>

<template>
	<div class="dropready_expand_wrapper">
		<div class="header_action">
			<div class="header_action_left_group">
				<button class="header_action_icon_btn return_btn"
					@click="handleWindowRestore"
					title="返回">
					<svg viewBox="0 0 16 16" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"
						stroke-linecap="round" stroke-linejoin="round">
						<polyline points="10 13 5 8 10 3"></polyline>
					</svg>
				</button>

				<div class="header_action_divider"></div>

				<button class="header_action_icon_btn other_btn"
					@click="handleAction('moveSelected')"
					title="剪切选中">
					<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.5"
						stroke-linecap="round" stroke-linejoin="round">
						<circle cx="6" cy="6" r="3"></circle>
						<circle cx="6" cy="18" r="3"></circle>
						<line x1="20" y1="4" x2="8.12" y2="15.88"></line>
						<line x1="14.47" y1="14.48" x2="20" y2="20"></line>
						<line x1="8.12" y1="8.12" x2="12" y2="12"></line>
					</svg>
				</button>

				<button class="header_action_icon_btn other_btn"
					@click="handleAction('copyMoveSelected')"
					title="复制选中">
					<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.5"
						stroke-linecap="round" stroke-linejoin="round">
						<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
						<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
					</svg>
				</button>


				<button class="header_action_icon_btn other_btn danger"
					@click="handleAction('deleteSelected')"
					title="删除选中">
					<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.5"
						stroke-linecap="round" stroke-linejoin="round">
						<path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2">
						</path>
						<line x1="10" y1="11" x2="10" y2="17"></line>
						<line x1="14" y1="11" x2="14" y2="17"></line>
					</svg>
				</button>

				<div class="header_action_divider"></div>

				<button class="header_action_icon_btn other_btn"
					@click="fileNameToggleSortType"
					:title="fileNameSortType === 'nameAsc' ? '切换为降序' : '切换为升序'">
					<svg v-if="fileNameSortType === 'nameAsc'" viewBox="0 0 24 24" width="16" height="16" fill="none" :stroke="'var(--accent)'" stroke-width="2">
						<path d="M3 4h13M3 8h9M3 12h5M19 20V4M15 8l4-4 4 4"></path>
					</svg>
					<svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" :stroke="'var(--accent)'" stroke-width="2">
						<path d="M3 4h13M3 8h9M3 12h5M19 4v16M15 16l4 4 4-4"></path>
					</svg>
				</button>
			</div>

			<div class="header_action_right_info">
				<span class="count_badge">共 {{ images.length }} 项</span>
			</div>
		</div>

		<div class="files_stack_area"
			:class="{ 'is_dragging': isDragging }"
			@dragenter="handleDragEnter"
			@dragleave="handleDragLeave"
			@dragover="handleDragOver"
			@drop="handleDrop">
			<div :class="['files_stack', filesStackViewMode]">
				<div v-for="img in fileNameSorted" class="file_item"
					:key="img.path"
					:class="{ 'is_selected': fileItemSelectedPaths.has(img.path) }"
					@click="fileItemToggleSelect(img.path)">
					<div class="file_thumbnails">
						<img :src="img.base64" />
					</div>
					<div class="file_info">
						<span class="file_name">{{ getFileNameFromImg(img.path) }}</span>
					</div>
				</div>
			</div>
			<div v-if="isDragging" class="drag_overlay">
			</div>
		</div>
	</div>
</template>

<style scoped>
.dropready_expand_wrapper {
	width: 100%;
	height: 100%;
	display: flex;
	flex-direction: column;
}

.header_action {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-top: 5px;
	margin-bottom: 5px;
	padding: 0px 0px 10px 0px;
	border-bottom: 1px solid var(--divider);
	flex-shrink: 0;
}

.header_action_left_group {
	display: flex;
	align-items: center;
	gap: 8px;
}

/* 通用图标按钮 */
.header_action_icon_btn {
	width: 24px;
	height: 24px;
	background-color: transparent;
	border: none;
	color: var(--text-muted);
	display: flex;
	align-items: center;
	justify-content: center;
	transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
	cursor: pointer;
}
.header_action_icon_btn:active {
	transform: scale(0.9);
}
.header_action_icon_btn svg {
	flex-shrink: 0;
	display: block; /* 消除 SVG 作为内联元素可能产生的底部间隙 */
}

/* 返回按钮 */
.return_btn {
	background: var(--bg-soft);
	border: 1px solid var(--border-soft);
	border-radius: 8px;
}
.return_btn:hover {
	background: var(--bg-soft-hover);
	border-color: var(--border-strong);
	color: var(--text-main);
}

/* 其他按钮 */
.other_btn:hover {
	color: var(--text-main);
}
.other_btn.danger:hover {
	color: var(--danger);
}

.header_action_divider {
	width: 1px;
	height: 15px;
	background: var(--border-soft);
	margin: 0 4px;
}

.header_action_right_info {
	font-size: 11px;
	color: var(--text-faint);
	font-weight: 400;
}

.count_badge {
	padding: 2px 8px;
	background: var(--bg-soft);
	border-radius: 10px;
}

.files_stack_area {
	min-height: 0;
	padding: 0px;
	/* background-color: aqua; */
	flex: 1;
	display: flex;
	align-items: center;
	justify-content: center;
	position: relative;
	overflow: hidden; /* 确保子元素不超出此边界 */
}

/* 滚动容器 */
.files_stack {
	width: 100%;
	height: 100%;
	padding: 5px;
	box-sizing: border-box;
	overflow-y: auto;
	overflow-x: hidden;
}

/* --- 自定义滚动条 (针对 Webkit 内核) --- */
.files_stack::-webkit-scrollbar {
	width: 5px;
}
.files_stack::-webkit-scrollbar-track {
	background: transparent;
}
.files_stack::-webkit-scrollbar-thumb {
	background: var(--border-soft);
}
.files_stack::-webkit-scrollbar-thumb:hover {
	background: var(--border-strong);
}

.file_thumbnails {
	position: relative;
	background-color: transparent;
	flex-shrink: 0; /* 防止在 Flex 布局中被压缩 */
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 4px;
	overflow: hidden; /* 必须：强制剪裁超出部分 */
}

.file_thumbnails img {
	width: 100%;
	height: 100%;
	object-fit: contain; /* 保持比例，在容器内缩放 */
	display: block; /* 消除底部间隙 */
}

/* 宫格布局 */
.files_stack.grid {
	width: 100%;
	height: 100%;
	padding: 10px 5px;
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
	grid-auto-rows: max-content;
	gap: 12px;
}

.grid .file_item {
	padding: 10px 5px;
	border: 1px solid transparent;
	border-radius: 4px;
	display: flex;
	flex-direction: column;
	align-items: center;
}

.grid .file_thumbnails {
	width: 64px;
	height: 64px;
	margin-bottom: 8px;
	background: transparent;
	border-radius: 6px;
}

.grid .file_name {
	width: 100%;
	display: -webkit-box;
	text-align: center;
	-webkit-line-clamp: 1;
	-webkit-box-orient: vertical;
	word-break: break-all; /* 强制在任何字符间换行，处理长文件名 */
	font-size: 11px;
	font-weight: 400;
	line-height: 1.4;
	overflow: hidden;
	overflow-wrap: anywhere; /* 进一步确保长字符串会在容器边缘折断 */
}
/* 通用项样式 */
.file_item:hover {
	background: var(--bg-soft-hover);
	cursor: pointer;
}
.file_item.is_selected {
	background: color-mix(in srgb, var(--accent) 14%, transparent);
	border-color: color-mix(in srgb, var(--accent) 34%, transparent);
}
.file_item.is_selected .file_name {
	color: var(--accent);
}

/* 拖拽覆盖层 */
.drag_overlay {
	position: absolute;
	inset: 0;
	z-index: 2005;
	border-radius: 10px;
	border: 2px dashed var(--border-strong);
	background: var(--bg-soft);
	display: flex;
	align-items: center;
	justify-content: center;
	pointer-events: none; /* 允许事件穿透到父级 */
}

.footer_info_area {
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0; /* 禁止 footer 区域收缩，保证其高度固定 */
}
</style>
