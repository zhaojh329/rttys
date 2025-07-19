<template>
  <RttySplitter :devid="devid" :config="rootConfig" @split="handleSplitPanel" @resize="handleResize" class="splitter-root"/>
  <div id="terminal-pool">
    <RttyTerm v-for="id in terms" :key="id" :data-terminal-id="id" :devid="devid" :panel-id="id" @split="handleSplitPanel"/>
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
  direction: 'horizontal',
  panels: [
    { id: paneRootID }
  ]
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

function moveTerminalsToPool() {
  const pool = document.getElementById('terminal-pool')

  terms.value.forEach(termId => {
    const term = document.querySelector(`[data-terminal-id="${termId}"]`)
    pool.appendChild(term)
  })
}

function moveTerminalsToPlaceholders() {
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

function handleResize() {
  dispatchEventRttyResize()
}

window.addEventListener('resize', handleResize)

function dispatchEventRttyResize() {
  window.dispatchEvent(new CustomEvent('rtty-resize'))
}

function handleSplitPanel(panelId, direction) {
  moveTerminalsToPool()

  splitPanel(rootConfig.value, panelId, 0, direction)

  moveTerminalsToPlaceholders()
}

function splitPanel(config, panelId, index, direction, parent) {
  if (config.id === panelId) {
    const newId = 'panel-' + Math.random().toString(36).substring(2, 10)
    if (parent.direction === direction || parent.panels.length < 2) {
      parent.direction = direction
      parent.panels.splice(index + 1, 0, { id: newId })
    } else {
      config.panels = [
        { id: config.id },
        { id: newId }
      ]
      config.direction = direction
      delete config.id
    }
    return true
  } else if (config.panels) {
    return config.panels.some((panel, i) => splitPanel(panel, panelId, i, direction, config))
  }
  return false
}
</script>

<style scoped>
.splitter-root {
  height: 100vh;
}
</style>
