<template>
  <div ref="content" class="content" :style="{top: axis.y + 'px', left: axis.x + 'px'}" v-if="model">
    <template v-for="(item, index) in menus" :key="item.name">
      <a @click="onMenuClick(item.name)" :style="{'text-decoration': item.underline ? 'underline' : 'none'}">
        {{item.caption || item.name}}
      </a>
      <hr v-if="index < menus.length - 1"/>
    </template>
  </div>
</template>

<script setup>
import { reactive, watch, nextTick, onBeforeUnmount, useTemplateRef } from 'vue'

defineProps({
  menus: Array
})

const model = defineModel()

const emit = defineEmits(['click'])

const content = useTemplateRef('content')
const axis = reactive({ x: 0, y: 0 })

const close = (e) => {
  const el = content.value

  if (e.clientX >= axis.x && e.clientX <= axis.x + el.clientWidth &&
    e.clientY >= axis.y && e.clientY <= axis.y + el.clientHeight) {
    return
  }

  model.value = null
}

const show = (clientX, clientY) => {
  document.addEventListener('mousedown', close)

  axis.x = clientX
  axis.y = clientY

  nextTick(() => {
    const el = content.value
    if (!el) return

    const rect = el.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let x = clientX
    let y = clientY

    if (x + rect.width > viewportWidth) {
      x = viewportWidth - rect.width - 15
    }

    if (y + rect.height > viewportHeight) {
      y = viewportHeight - rect.height - 15
    }

    x = Math.max(15, x)
    y = Math.max(15, y)

    axis.x = x
    axis.y = y
  })
}

const onMenuClick = (name) => {
  model.value = null
  emit('click', name)
}

watch(() => model.value, (val) => {
  if (!val)
    document.removeEventListener('mousedown', close)
  else
    show(val.x, val.y)
})

onBeforeUnmount(() => document.removeEventListener('mousedown', close))
</script>

<style scoped>
  .content {
    position: fixed;
    z-index: 9999;
    background-color: #f9f9f9;
    min-width: 160px;
    border-radius: 5px;
    padding: 5px;
    box-shadow: 0px 8px 16px 0px rgba(0, 0, 0, 0.2);
  }

  .content a {
    color: black;
    padding: 5px 16px;
    text-decoration: none;
    display: block;
  }

  .content a:hover {
    background-color: #90C8F6;
    cursor: default;
  }

  .content hr {
    margin: 0;
    border: 1px solid #c1bcbc;
  }
</style>
