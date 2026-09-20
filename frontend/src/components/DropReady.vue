<script setup>
import { useDragState } from '../composables/useDragState.js'

const emit = defineEmits(['windowExpand'])
const { isDragging, handleDragEnter, handleDragLeave, handleDragOver, handleDrop } = useDragState()

const props = defineProps({
	images: {
		type: Array,
		default: () => []
	},
	totalCount: {
		type: Number,
		default: 0
	},
})

// 尺寸调整与居中统一交由 App 层的 useWindowResize 处理，避免魔数与重复调用
const handleWindowExpand = () => {
	emit('windowExpand')
}
</script>

<template>
	<div class="dropready_wrapper"
		:class="{ 'is_dragging': isDragging }"
		@dragenter="handleDragEnter"
		@dragleave="handleDragLeave"
		@dragover="handleDragOver"
		@drop="handleDrop"
		draggable="true">
		<div class="minimap_stack_area">
			<div class="minimap_stack">
				<div v-for="(img, index) in images"
					:key="index"
					:class="['minimap_card', `is-${img.orientation}`]"
					:style="{
						'--index': index,
						'--total': images.length,
						'aspect-ratio': img.ratio
					}">
					<div class="minimap_inner">
						<img :src="img.base64"
							draggable="false" />
					</div>
				</div>
			</div>
			<div v-if="isDragging" class="drag_overlay">
			</div>
		</div>

		<div class="footer_action">
			<button class="count_content"
				@click="handleWindowExpand">
				<span>{{ totalCount || images.length }} 项内容已选中</span>
				<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"
					stroke-linecap="round" stroke-linejoin="round">
					<polyline points="15 3 21 3 21 9"></polyline>
					<polyline points="9 21 3 21 3 15"></polyline>
					<line x1="21" y1="3" x2="14" y2="10"></line>
					<line x1="3" y1="21" x2="10" y2="14"></line>
				</svg>
			</button>
		</div>
	</div>
</template>

<style scoped>
.dropready_wrapper {
	width: 100%;
	height: 100%;
	display: flex;
	flex-direction: column;
}

.minimap_stack_area {
	min-height: 50px;
	padding: 0px 0px 10px 0px;
	flex: 1;
	display: flex;
	align-items: center;
	justify-content: center;
	position: relative;
	cursor: grabbing;
}

.minimap_stack {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	position: relative;
	overflow: hidden; /* 确保内部旋转的边缘不会撑开父级 */
}

.minimap_card {
	max-width: 75%;
	max-height: 85%;
	background-color: var(--bg-surface-1);
	padding: 3px 3px 4px 3px;
	border-radius: 4px;
	box-shadow: 0 5px 15px var(--shadow-soft);
	z-index: calc(var(--total) - var(--index));
	transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1); /* 堆叠偏移算法 */
	transform:
		translate(-50%, -50%) /* 自身居中 */
		rotate(calc(var(--index) * -3deg + 1.5deg)) /* 叠加旋转 */
		translate(calc(var(--index) * -0px), calc(var(--index) * -0px)); /* 叠加堆叠偏移 */
	position: absolute;
	top: 50%; /* 将定位基准设置到中心 */
	left: 50%; /* 将定位基准设置到中心 */
}
.minimap_card:hover {
	z-index: 100;
	background-color: var(--bg-surface-1);
	transform:
		translate(-50%, -50%) /* 保持居中基准 */
		translate(0, -5px) /* 向上浮动效果 */
		scale(1.01) /* 放大 */
		rotate(0deg) !important; /* 复位旋转 */
	box-shadow: 0 10px 30px var(--shadow-strong);
}

.minimap_inner {
	width: 100%;
	height: 100%;
	border-radius: 2px;
	overflow: hidden;
}

.minimap_inner img {
	width: 100%;
	height: 100%;
	object-fit: cover; /* 确保填满内框 */
}

/* 拖拽覆盖层 */
.drag_overlay {
	position: absolute;
	inset: 0;
	z-index: 2005;
	background: var(--bg-soft);
	border-radius: 10px;
	border: 2px dashed var(--border-strong);
	display: flex;
	align-items: center;
	justify-content: center;
	pointer-events: none; /* 允许事件穿透到父级 */
}

.footer_action {
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0; /* 禁止 footer 区域收缩，保证其高度固定 */
}

.count_content {
	--wails-draggable: no-drag;
	width: max-content;
	min-width: 120px;
	max-width: 170px;
	padding: 8px 16px;
	background: var(--bg-soft);
	border: 1px solid var(--border-soft);
	border-radius: 40px;
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 5px;
	color: var(--text-main);
	font-size: 13px;
	font-weight: 500;
	cursor: pointer;
}
.count_content:hover {
	background: var(--bg-soft-hover);
	border-color: var(--border-strong);
	color: var(--text-main);
	font-weight: 600;
}

.count_content span {
	display: flex; /* 确保 span 内部也遵循 flex 对齐 */
	align-items: center;
	line-height: 1; /* 消除默认行高导致的垂直偏差 */
	transform: translateY(-0.5px); /* 微调：大多数字体在 flex 垂直居中时视觉上会偏下约 0.5px，进行补偿 */
	/* letter-spacing: 0.5px; /* 微增字间距提升可读性 */
}

.count_content svg {
	flex-shrink: 0;
	display: block; /* 消除 SVG 作为内联元素可能产生的底部间隙 */
}
</style>
