<template>
  <div class="keyboard-area"
    @mousedown="startDrag" @mousemove="onDrag" @mouseup="stopDrag" @mouseleave="stopDrag"
    @touchstart="startDrag" @touchmove="onDrag" @touchend="stopDrag">
    <div class="drag-handle">
      <span class="drag-icon">⋮⋮</span>
      <span class="keyboard-title">{{ $t('Virtual Keyboard') }}</span>
      <button class="close-btn" @click="closeKeyboard" @mousedown.stop @touchstart.stop>✕</button>
    </div>
    <div ref="simple-keyboard" class="simple-keyboard"></div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, useTemplateRef } from 'vue'
import Keyboard from 'simple-keyboard'
import 'simple-keyboard/build/css/index.css'

const emit = defineEmits(['keypress', 'close'])

const simpleKeyboard = useTemplateRef('simple-keyboard')

let keyboard = null
const shiftPressed = ref(false)
const ctrlPressed = ref(false)
const altPressed = ref(false)
const capsLockPressed = ref(false)

const isDragging = ref(false)
const dragStart = reactive({ x: 0, y: 0 })
const keyboardPosition = reactive({ x: 0, y: 0 })
const initKeyboard = () => {
  keyboard = new Keyboard(simpleKeyboard.value, {
    onKeyPress: button => onKeyPress(button),
    layoutName: 'default',
    layout: {
      default: [
        '{escape} ` 1 2 3 4 5 6 7 8 9 0 - = {backspace}',
        '{tab} q w e r t y u i o p [ ]',
        '{capslock} a s d f g h j k l ; \'',
        '{shift} z x c v b n m , . / \\',
        '{ctrl} {alt} {space} {enter}'
      ],
      shift: [
        '{escape} ~ ! @ # $ % ^ & * ( ) _ + {backspace}',
        '{tab} Q W E R T Y U I O P { }',
        '{capslock} A S D F G H J K L : "',
        '{shift} Z X C V B N M < > ? |',
        '{ctrl} {alt} {space} {enter}'
      ],
      capslock: [
        '{escape} ` 1 2 3 4 5 6 7 8 9 0 - = {backspace}',
        '{tab} Q W E R T Y U I O P [ ]',
        '{capslock} A S D F G H J K L ; \'',
        '{shift} Z X C V B N M , . / \\',
        '{ctrl} {alt} {space} {enter}'
      ]
    },
    display: {
      '{escape}': 'Esc',
      '{backspace}': '⌫',
      '{enter}': '↵',
      '{shift}': '⇧',
      '{tab}': '⇥',
      '{space}': '␣',
      '{capslock}': '⇪',
      '{ctrl}': 'Ctrl',
      '{alt}': 'Alt'
    },
    buttonTheme: [
      {
        class: 'hg-red',
        buttons: '{backspace} {enter} {shift} {tab} {capslock}'
      },
      {
        class: 'hg-blue',
        buttons: '{ctrl} {alt}'
      }
    ],
    mergeDisplay: true,
    syncInstanceInputs: true,
    physicalKeyboardHighlight: true,
    physicalKeyboardHighlightTextColor: 'white',
    physicalKeyboardHighlightBgColor: '#9ab4d0'
  })
}

const onKeyPress = (button) => {
  if (button === '{shift}') {
    shiftPressed.value = !shiftPressed.value
    updateKeyboardLayout()
    updateModifierKeyStyles()
    return
  }

  if (button === '{ctrl}') {
    ctrlPressed.value = !ctrlPressed.value
    updateModifierKeyStyles()
    return
  }

  if (button === '{alt}') {
    altPressed.value = !altPressed.value
    updateModifierKeyStyles()
    return
  }

  if (button === '{capslock}') {
    capsLockPressed.value = !capsLockPressed.value
    updateKeyboardLayout()
    updateModifierKeyStyles()
    return
  }
  const specialKeys = {
    '{enter}': '\r',
    '{space}': ' ',
    '{tab}': '\t',
    '{backspace}': '\x7f',
    '{escape}': '\x1b',
    '{arrowup}': '\x1b[A',
    '{arrowdown}': '\x1b[B',
    '{arrowleft}': '\x1b[D',
    '{arrowright}': '\x1b[C',
    '{home}': '\x1b[H',
    '{end}': '\x1b[F',
    '{pageup}': '\x1b[5~',
    '{pagedown}': '\x1b[6~',
    '{insert}': '\x1b[2~'
  }

  let keyToSend = ''

  if (specialKeys[button]) {
    keyToSend = specialKeys[button]
  } else if (button.length === 1) {
    keyToSend = button

    if (ctrlPressed.value) {
      const char = button.toLowerCase()
      if (char >= 'a' && char <= 'z') {
        const ctrlCode = char.charCodeAt(0) - 96
        keyToSend = String.fromCharCode(ctrlCode)
      } else {
        const ctrlMap = {
          '[': '\x1b', // Ctrl+[
          ']': '\x1d', // Ctrl+]
          '\\': '\x1c', // Ctrl+\
          '/': '\x1f', // Ctrl+/
          ' ': '\x00' // Ctrl+Space
        }
        if (ctrlMap[char]) {
          keyToSend = ctrlMap[char]
        }
      }
    } else if (altPressed.value) {
      keyToSend = '\x1b' + button
    }
  } else {
    return
  }

  emit('keypress', keyToSend)

  ctrlPressed.value = false
  altPressed.value = false
  updateModifierKeyStyles()
}

const updateKeyboardLayout = () => {
  if (keyboard) {
    let layoutName = 'default'

    if (shiftPressed.value && capsLockPressed.value) {
      layoutName = 'shift'
    } else if (shiftPressed.value) {
      layoutName = 'shift'
    } else if (capsLockPressed.value) {
      layoutName = 'capslock'
    }

    keyboard.setOptions({
      layoutName: layoutName
    })
  }
}

const updateModifierKeyStyles = () => {
  if (!keyboard) return

  const shiftButtons = keyboard.getButtonElement('{shift}')
  const ctrlButtons = keyboard.getButtonElement('{ctrl}')
  const altButtons = keyboard.getButtonElement('{alt}')
  const capsLockButtons = keyboard.getButtonElement('{capslock}')

  if (shiftButtons) {
    const buttons = Array.isArray(shiftButtons) ? shiftButtons : [shiftButtons]
    buttons.forEach(btn => {
      if (shiftPressed.value) {
        btn.classList.add('hg-shiftActive')
        btn.classList.remove('hg-activeButton')
      } else {
        btn.classList.remove('hg-shiftActive')
      }
    })
  }

  if (ctrlButtons) {
    const buttons = Array.isArray(ctrlButtons) ? ctrlButtons : [ctrlButtons]
    buttons.forEach(btn => {
      if (ctrlPressed.value) {
        btn.classList.add('hg-ctrlActive')
        btn.classList.remove('hg-activeButton')
      } else {
        btn.classList.remove('hg-ctrlActive')
      }
    })
  }

  if (altButtons) {
    const buttons = Array.isArray(altButtons) ? altButtons : [altButtons]
    buttons.forEach(btn => {
      if (altPressed.value) {
        btn.classList.add('hg-altActive')
        btn.classList.remove('hg-activeButton')
      } else {
        btn.classList.remove('hg-altActive')
      }
    })
  }

  if (capsLockButtons) {
    const buttons = Array.isArray(capsLockButtons) ? capsLockButtons : [capsLockButtons]
    buttons.forEach(btn => {
      if (capsLockPressed.value) {
        btn.classList.add('hg-capslockActive')
        btn.classList.remove('hg-activeButton')
      } else {
        btn.classList.remove('hg-capslockActive')
      }
    })
  }
}

const destroyKeyboard = () => {
  if (keyboard) {
    keyboard.destroy()
    keyboard = null
  }
}

const getEventPosition = (e) => {
  if (e.type.startsWith('touch')) {
    const touch = e.touches[0] || e.changedTouches[0]
    return { x: touch.clientX, y: touch.clientY }
  } else {
    return { x: e.clientX, y: e.clientY }
  }
}

const startDrag = (e) => {
  const target = e.target.closest ? e.target.closest('.drag-handle') : null
  if (target) {
    const keyboardEl = e.currentTarget

    isDragging.value = true

    if (keyboardPosition.x === 0 && keyboardPosition.y === 0) {
      const rect = keyboardEl.getBoundingClientRect()
      const parentRect = keyboardEl.parentElement.getBoundingClientRect()
      keyboardPosition.x = rect.left - parentRect.left
      keyboardPosition.y = rect.top - parentRect.top
      keyboardEl.style.left = keyboardPosition.x + 'px'
      keyboardEl.style.top = keyboardPosition.y + 'px'
      keyboardEl.style.bottom = 'auto'
      keyboardEl.style.transform = 'none'
    }

    const pos = getEventPosition(e)
    dragStart.x = pos.x - keyboardPosition.x
    dragStart.y = pos.y - keyboardPosition.y
    e.preventDefault && e.preventDefault()
  }
}

const onDrag = (e) => {
  if (isDragging.value) {
    const pos = getEventPosition(e)
    keyboardPosition.x = pos.x - dragStart.x
    keyboardPosition.y = pos.y - dragStart.y
    const keyboardEl = e.currentTarget
    keyboardEl.style.left = keyboardPosition.x + 'px'
    keyboardEl.style.top = keyboardPosition.y + 'px'
    e.preventDefault && e.preventDefault()
  }
}

const stopDrag = () => isDragging.value = false

const closeKeyboard = () => emit('close')

onMounted(() => initKeyboard())
onUnmounted(() => destroyKeyboard())
</script>

<style scoped>
  .keyboard-area {
    background-color: rgba(248, 249, 250, 0.95);
    border-radius: 12px;
    padding: 0;
    width: 100%;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.2);
    overflow: hidden;
  }

  .drag-handle {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 8px 16px;
    cursor: move;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    font-weight: 600;
    user-select: none;
  }

  .drag-icon {
    font-size: 14px;
    opacity: 0.8;
  }

  .keyboard-title {
    flex: 1;
    text-align: center;
  }

  .close-btn {
    background: rgba(255, 255, 255, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.3);
    border-radius: 4px;
    color: white;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    font-size: 12px;
    font-weight: bold;
    transition: all 0.2s ease;
    padding: 0;
  }

  .close-btn:hover {
    background: rgba(255, 255, 255, 0.3);
    transform: scale(1.1);
  }

  .close-btn:active {
    transform: scale(0.95);
  }

  .simple-keyboard {
    margin: 0;
    background-color: transparent;
    border-radius: 0;
    box-shadow: none;
    width: 100%;
  }

  :deep(.simple-keyboard .hg-button) {
    background: #ffffff;
    border: 1px solid #ddd;
    border-radius: 4px;
    color: #333;
    font-weight: 500;
    font-size: clamp(1rem, 2.5vw, 1.5rem);
    min-height: clamp(25px, 3vw, 32px);
    transition: all 0.1s ease;
  }

  :deep(.simple-keyboard .hg-button:hover) {
    background: #e9ecef;
    border-color: #adb5bd;
  }

  :deep(.simple-keyboard .hg-button:active) {
    background: #dee2e6;
    transform: scale(0.98);
  }

  :deep(.simple-keyboard .hg-red) {
    background: #dc3545;
    color: white;
    border-color: #dc3545;
  }

  :deep(.simple-keyboard .hg-red:hover) {
    background: #c82333;
    border-color: #bd2130;
  }

  :deep(.simple-keyboard .hg-blue) {
    background: #007bff;
    color: white;
    border-color: #007bff;
  }

  :deep(.simple-keyboard .hg-blue:hover) {
    background: #0056b3;
    border-color: #004085;
  }

  :deep(.simple-keyboard .hg-activeButton) {
    background: #28a745 !important;
    color: white !important;
    border-color: #28a745 !important;
    box-shadow: 0 0 0 2px rgba(40, 167, 69, 0.25);
  }

  :deep(.simple-keyboard .hg-shiftActive) {
    background: #fd7e14 !important;
    color: white !important;
    border-color: #fd7e14 !important;
    box-shadow: 0 0 0 2px rgba(253, 126, 20, 0.25);
    animation: shiftPulse 1.5s infinite;
  }

  :deep(.simple-keyboard .hg-capslockActive) {
    background: #6f42c1 !important;
    color: white !important;
    border-color: #6f42c1 !important;
    box-shadow: 0 0 0 2px rgba(111, 66, 193, 0.25);
    animation: capslockPulse 2s infinite;
  }

  :deep(.simple-keyboard .hg-ctrlActive) {
    background: #28a745 !important;
    color: white !important;
    border-color: #28a745 !important;
    box-shadow: 0 0 0 2px rgba(40, 167, 69, 0.25);
    animation: ctrlPulse 1s infinite;
  }

  :deep(.simple-keyboard .hg-altActive) {
    background: #17a2b8 !important;
    color: white !important;
    border-color: #17a2b8 !important;
    box-shadow: 0 0 0 2px rgba(23, 162, 184, 0.25);
    animation: altPulse 1.2s infinite;
  }

  @keyframes shiftPulse {
    0%, 100% {
      box-shadow: 0 0 0 2px rgba(253, 126, 20, 0.25);
    }
    50% {
      box-shadow: 0 0 0 4px rgba(253, 126, 20, 0.4);
    }
  }

  @keyframes capslockPulse {
    0%, 100% {
      box-shadow: 0 0 0 2px rgba(111, 66, 193, 0.25);
    }
    50% {
      box-shadow: 0 0 0 4px rgba(111, 66, 193, 0.4);
    }
  }

  @keyframes ctrlPulse {
    0%, 100% {
      box-shadow: 0 0 0 2px rgba(40, 167, 69, 0.25);
    }
    50% {
      box-shadow: 0 0 0 4px rgba(40, 167, 69, 0.4);
    }
  }

  @keyframes altPulse {
    0%, 100% {
      box-shadow: 0 0 0 2px rgba(23, 162, 184, 0.25);
    }
    50% {
      box-shadow: 0 0 0 4px rgba(23, 162, 184, 0.4);
    }
  }
</style>
