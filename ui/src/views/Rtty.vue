<template>
  <RttySplitter :devid="devid" :config="rootConfig" @split="handleSplitPanel" @close="handleClosePanel" @resize="handleResize" class="splitter-root"/>
  <div id="terminal-pool">
    <RttyTerm v-for="id in terms" :key="id" :data-terminal-id="id" :devid="devid" :panel-id="id" @split="handleSplitPanel" @close="handleClosePanel"/>
  </div>
</template>

<script setup>
import RttySplitter from '../components/RttySplitter.vue'
import RttyTerm from '../components/RttyTerm.vue'
import { ref, computed, onMounted, nextTick } from 'vue'

defineProps({
  devid: {
    type: String,
    required: true
  }
})

const paneRootID = 'rtty-panel-root'

const rootConfig = ref({
  id: paneRootID
})

const terms = computed(() => {
  const ids = []
  const traverse = (node) => {
    if (node.id) {
      ids.push(node.id)
    } else if (node.panels) {
      node.panels.forEach(traverse)
    }
  }
  traverse(rootConfig.value)
  return ids
})

const moveTerminalsToPool = () => {
  const pool = document.getElementById('terminal-pool')

  terms.value.forEach(termId => {
    const term = document.querySelector(`[data-terminal-id="${termId}"]`)
    pool.appendChild(term)
  })
}

const moveTerminalsToPlaceholders = () => {
  nextTick(() => {
    terms.value.forEach(termId => {
      const term = document.querySelector(`[data-terminal-id="${termId}"]`)
      const placeholder = document.getElementById(termId)
      placeholder.appendChild(term)
    })

    setTimeout(() => dispatchEventRttyResize(), 100)
  })
}

onMounted(() => moveTerminalsToPlaceholders())

const handleResize = () => dispatchEventRttyResize()

window.addEventListener('resize', handleResize)

const dispatchEventRttyResize = () => window.dispatchEvent(new CustomEvent('rtty-resize'))

const handleClosePanel = (panelId) => {
  moveTerminalsToPool()
  deletePanel(rootConfig.value, panelId, 0)
  moveTerminalsToPlaceholders()
}

const deletePanel = (config, panelId, index, parent) => {
  if (parent && config.id === panelId) {
    parent.panels.splice(index, 1)
    if (parent.panels.length === 1) {
      parent.id = parent.panels[0].id
      delete parent.direction
      delete parent.panels
    }
    return true
  } else if (config.panels) {
    return config.panels.some((panel, i) => deletePanel(panel, panelId, i, config))
  }
  return false
}

const handleSplitPanel = (panelId, direction) => {
  moveTerminalsToPool()
  splitPanel(rootConfig.value, panelId, 0, direction)
  moveTerminalsToPlaceholders()
}

const splitPanel = (config, panelId, index, position, parent) => {
  if (config.id === panelId) {
    const direction = (position === 'left' || position === 'right') ? 'horizontal' : 'vertical'
    const newId = 'panel-' + Math.random().toString(36).substring(2, 10)

    if (parent && (parent.direction === direction || parent.panels.length < 2)) {
      parent.direction = direction
      if (position === 'right' || position === 'down')
        index++
      parent.panels.splice(index, 0, { id: newId })
    } else {
      if (position === 'right' || position === 'down') {
        config.panels = [
          { id: config.id },
          { id: newId }
        ]
      } else {
        config.panels = [
          { id: newId },
          { id: config.id }
        ]
      }
      config.direction = direction
      delete config.id
    }
    return true
  } else if (config.panels) {
    return config.panels.some((panel, i) => splitPanel(panel, panelId, i, position, config))
  }
  return false
}
</script>

<style scoped>
.splitter-root {
  height: 100vh;
}
</style>
