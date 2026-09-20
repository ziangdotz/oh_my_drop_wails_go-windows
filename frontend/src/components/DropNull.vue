<script setup>
import { useDragState } from '../composables/useDragState.js'

const { isDragging, handleDragEnter, handleDragLeave, handleDragOver, handleDrop } = useDragState()
</script>

<template>
	<div class="dropnull_wrapper"
		:class="{ 'dragging': isDragging }"
		@dragenter="handleDragEnter"
		@dragleave="handleDragLeave"
		@dragover="handleDragOver"
		@drop="handleDrop">
		<p>{{ isDragging ? '松开放置' : '拖放到此处' }}</p>
	</div>
</template>

<style scoped>
.dropnull_wrapper {
	width: 100%;
	height: 100%;
	margin: 0px;
	box-sizing: border-box;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	border: 1px dashed var(--border-soft);
	border-radius: 8px;
	color: var(--text-faint);
	font-size: 14px;
	font-weight: 500;
	transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
	pointer-events: auto; /* 确保能响应事件 */
}
.dropnull_wrapper:hover {
	background-color: var(--bg-soft);
	border: 2px dashed var(--border-strong);
	cursor: no-drop
}
/* 确保内部文字不触发 dragleave 事件 */
.dropnull_wrapper * {
	pointer-events: none;
}

/* 当文件拖入时的样式 */
.dropnull_wrapper.dragging {
	background-color: var(--bg-soft-hover);
	border: 2px dashed var(--border-strong);
	color: var(--text-main);
	font-weight: 600;
	cursor: grab;
	transform: scale(0.9); /* 增加缩放微动效，更有"吸入感" */
}
</style>
