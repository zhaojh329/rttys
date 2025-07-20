<template>
  <div class="splitter-panel-root">
    <div v-if="!config.panels" :key="config.id" :id="config.id" class="rtty-placeholder"></div>
    <div v-else class="rtty-splitter" :class="config.direction">
      <div v-for="(panel, index) in config.panels" :key="panel.id || `panel-${index}`" class="splitter-panel" :style="getPanelStyle(index)">
        <RttySplitter :config="panel" :devid="devid" @split="handleSplitPanel" @close="handleClosePanel" @resize="handleResize"/>
        <div v-if="index < config.panels.length - 1" class="splitter-bar" :class="config.direction" @mousedown="startResize(index, $event)"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  devid: {
    type: String,
    required: true
  },
  config: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['split', 'close', 'resize'])

const panelSizes = ref([])
const isResizing = ref(false)
const resizingIndex = ref(-1)
const startPos = ref(0)
const startSizes = ref([])
const containerRef = ref(null)

const initializePanelSizes = () => {
  if (!props.config.panels) return

  if (panelSizes.value.length !== props.config.panels.length) {
    const defaultSize = 100 / props.config.panels.length
    panelSizes.value = new Array(props.config.panels.length).fill(defaultSize)
  }
}

const getPanelStyle = (index) => {
  if (!props.config.panels) return {}

  initializePanelSizes()

  const size = panelSizes.value[index] || (100 / props.config.panels.length)

  if (props.config.direction === 'horizontal') {
    return {
      width: `${size}%`,
      height: '100%',
      position: 'relative'
    }
  } else {
    return {
      height: `${size}%`,
      width: '100%',
      position: 'relative'
    }
  }
}

const startResize = (index, event) => {
  isResizing.value = true
  resizingIndex.value = index
  startPos.value = props.config.direction === 'horizontal' ? event.clientX : event.clientY
  startSizes.value = [...panelSizes.value]
  containerRef.value = event.target.closest('.rtty-splitter')

  document.addEventListener('mousemove', handleResizeMove)
  document.addEventListener('mouseup', stopResize)

  event.preventDefault()
  document.body.style.userSelect = 'none'
}

const handleResizeMove = (event) => {
  if (!isResizing.value) return

  const currentPos = props.config.direction === 'horizontal' ? event.clientX : event.clientY
  const delta = currentPos - startPos.value

  const container = containerRef.value
  if (!container) return

  const containerSize = props.config.direction === 'horizontal' ? container.clientWidth : container.clientHeight

  const deltaPercent = (delta / containerSize) * 100

  const newSizes = [...startSizes.value]
  const leftIndex = resizingIndex.value
  const rightIndex = leftIndex + 1

  const minSize = 5
  const maxLeftDecrease = Math.max(0, newSizes[leftIndex] - minSize)
  const maxRightDecrease = Math.max(0, newSizes[rightIndex] - minSize)

  const actualDelta = Math.max(-maxLeftDecrease, Math.min(maxRightDecrease, deltaPercent))

  newSizes[leftIndex] = startSizes.value[leftIndex] + actualDelta
  newSizes[rightIndex] = startSizes.value[rightIndex] - actualDelta

  panelSizes.value = newSizes
}

const stopResize = () => {
  isResizing.value = false
  resizingIndex.value = -1
  containerRef.value = null

  document.removeEventListener('mousemove', handleResizeMove)
  document.removeEventListener('mouseup', stopResize)

  document.body.style.userSelect = ''

  emit('resize')
}

const handleSplitPanel = (panelId, position) => emit('split', panelId, position)
const handleClosePanel = (panelId) => emit('close', panelId)
const handleResize = () => emit('resize')
</script>

<style scoped>
.splitter-panel-root {
  width: 100%;
  height: 100%;
}

.rtty-placeholder {
  width: 100%;
  height: 100%;
}

.rtty-splitter {
  width: 100%;
  height: 100%;
  display: flex;
  position: relative;
}

.rtty-splitter.horizontal {
  flex-direction: row;
}

.rtty-splitter.vertical {
  flex-direction: column;
}

.splitter-panel {
  overflow: hidden;
  position: relative;
}

.splitter-bar {
  position: absolute;
  background-color: #dcdfe6;
  z-index: 10;
  transition: background-color 0.2s;
}

.splitter-bar:hover {
  background-color: #409eff;
}

.splitter-bar.horizontal {
  top: 0;
  right: -2px;
  width: 4px;
  height: 100%;
  cursor: ew-resize;
}

.splitter-bar.vertical {
  bottom: -2px;
  left: 0;
  width: 100%;
  height: 4px;
  cursor: ns-resize;
}
</style>
