<template>
  <div class="terminal-container">
    <div ref="terminal" class="terminal" @contextmenu.prevent="showContextmenu"></div>
    <el-button v-show="isConnected && !showKeyboard" @click="toggleKeyboard" type="primary" size="small" circle class="keyboard-toggle-btn">⌨</el-button>
    <RttyKeyboard v-show="showKeyboard" @keypress="handleKeypress" @close="hideKeyboard" class="floating-keyboard"/>
    <el-dialog v-model="fileCtx.modal" :title="$t('Upload file to device')" @close="onUploadDialogClosed" :width="400">
      <el-upload :before-upload="beforeUpload" action="#">
        <el-button type="primary">{{ $t("Select file") }}</el-button>
      </el-upload>
      <p v-if="fileCtx.file !== null"> {{ fileCtx.file.name }}</p>
      <template #footer>
        <el-button @click="fileCtx.modal = false">{{ $t('Cancel') }}</el-button>
        <el-button type="primary" @click="doUploadFile">{{ $t('OK') }}</el-button>
      </template>
    </el-dialog>
    <ContextMenu v-model="contextmenuPos" :menus="contextmenus" @click="onContextmenuClick"/>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick, useTemplateRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElLoading, ElMessage, ElMessageBox } from 'element-plus'
import useClipboard from 'vue-clipboard3'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import OverlayAddon from '../xterm-addon/xterm-addon-overlay'
import ContextMenu from '../components/ContextMenu.vue'
import RttyKeyboard from '../components/RttyKeyboard.vue'

const LoginErrorOffline = 4000
const LoginErrorBusy = 4001
const LoginErrorTimeout = 4002

const MsgTypeFileData = 0x03

const ReadFileBlkSize = 63 * 1024

const AckBlkSize = 4 * 1024

const props = defineProps({
  devid: String,
  panelId: String
})

const emit = defineEmits(['split', 'close'])

const router = useRouter()
const { t } = useI18n()
const { toClipboard } = useClipboard()

const terminal = useTemplateRef('terminal')
const contextmenuPos = ref(null)

const contextmenus = [
  {name: 'copy', caption: t('Copy - Ctrl+Insert')},
  {name: 'paste', caption: t('Paste - Shift+Insert')},
  {name: 'clear', caption: t('Clear Scrollback')},
  {name: 'font+', caption: t('font+')},
  {name: 'font-', caption: t('font-')},
  {name: 'upload', caption: t('Upload file') + ' - rtty -R'},
  {name: 'download', caption: t('Download file') + ' - rtty -S file'},
  {name: 'split-left', caption: t('split-left')},
  {name: 'split-right', caption: t('split-right')},
  {name: 'split-up', caption: t('split-up')},
  {name: 'split-down', caption: t('split-down')},
  {name: 'close', caption: t('Close')},
  {name: 'about', caption: t('About')}
]

const fileCtx = reactive({
  modal: false,
  accepted: false,
  file: null,
  offset: 0,
  fr: new FileReader(),
  name: '',
  chunks: []
})

let disposables = []
let socket = null
let term = null
let fitAddon = null
let unack = 0
const showKeyboard = ref(false)
const isConnected = ref(false)

const copyText = async(text) => {
  try {
    await toClipboard(text)
    return Promise.resolve()
  } catch (err) {
    return Promise.reject(err)
  }
}

const showContextmenu = (e) => contextmenuPos.value = { x: e.clientX, y: e.clientY }

const toggleKeyboard = () => showKeyboard.value = !showKeyboard.value

const hideKeyboard = () => showKeyboard.value = false

const handleKeypress = (keyData) => sendTermData(keyData)

const onContextmenuClick = (name) => {
  if (name === 'copy') {
    const text = term.getSelection()
    if (text)
      copyText(text).then(() => ElMessage.success(t('Copied to clipboard')))
  } else if (name === 'paste') {
    pasteFromClipboard()
  } else if (name === 'clear') {
    term.clear()
  } else if (name === 'font+') {
    updateFontSize(1)
  } else if (name === 'font-') {
    updateFontSize(-1)
  } else if (name === 'upload') {
    ElMessage.success(t('Please execute command "rtty -R" in current terminal!'))
  } else if (name === 'download') {
    ElMessage.success(t('Please execute command "rtty -S file" in current terminal!'))
  } else if (name === 'split-left') {
    emit('split', props.panelId, 'left')
  } else if (name === 'split-right') {
    emit('split', props.panelId, 'right')
  } else if (name === 'split-up') {
    emit('split', props.panelId, 'up')
  } else if (name === 'split-down') {
    emit('split', props.panelId, 'down')
  } else if (name === 'close') {
    emit('close', props.panelId)
  } else if (name === 'about') {
    window.open('https://github.com/zhaojh329/rtty')
  }

  term.focus()
}

const pasteFromClipboard = async() => {
  try {
    if (!navigator.clipboard || !navigator.clipboard.readText) {
      ElMessage.info(t('Please use shortcut "Shift+Insert"'))
      return
    }

    const text = await navigator.clipboard.readText()
    if (text) {
      sendTermData(text)
      ElMessage.success(t('Pasted from clipboard'))
    }
  } catch (error) {
    if (error.name === 'NotAllowedError') {
      ElMessageBox.alert(t('clipboard_instructions'), t('Clipboard Permission Required'), {
        type: 'warning'
      })
    } else {
      ElMessage.info(t('Please use shortcut "Shift+Insert"'))
    }
  }
}

const updateFontSize = (size) => {
  term.options.fontSize += size
  fitAddon.fit()
}

const onUploadDialogClosed = () => {
  term.focus()
  if (fileCtx.accepted)
    return
  fileCtx.file = null
  const msg = {type: 'fileCanceled'}
  socket.send(JSON.stringify(msg))
}

const beforeUpload = (file) => {
  fileCtx.file = file
  return false
}

const sendFileInfo = (file) => {
  const msg = {type: 'fileInfo', size: file.size, name: file.name}
  socket.send(JSON.stringify(msg))
}

const readFileBlob = (fr, file, offset, size) => {
  const blob = file.slice(offset, offset + size)
  fr.readAsArrayBuffer(blob)
}

const doUploadFile = () => {
  if (!fileCtx.file) {
    onUploadDialogClosed()
    return
  }

  term.focus()

  if (fileCtx.file.size > 0xffffffff) {
    ElMessage.error(t('The file you will upload is too large(> 4294967295 Byte)'))
    return
  }

  fileCtx.accepted = true
  fileCtx.modal = false

  sendFileInfo(fileCtx.file)

  if (fileCtx.file.size === 0) {
    sendFileData(null)
    return
  }

  fileCtx.offset = 0

  const fr = fileCtx.fr

  fr.onload = e => {
    fileCtx.offset += e.loaded
    sendFileData(new Uint8Array(fr.result))
  }
  readFileBlob(fr, fileCtx.file, fileCtx.offset, ReadFileBlkSize)
}

const sendTermData = (data) => socket.send(new Uint8Array([0, ...new TextEncoder().encode(data)]))

const sendFileData = (data) => {
  let b

  if (data !== null)
    b = new Uint8Array([1, MsgTypeFileData, ...data])
  else
    b = new Uint8Array([1, MsgTypeFileData])

  socket.send(b)
}

const fitTerm = () => nextTick(() => fitAddon.fit())

const closed = () => {
  if (term)
    term.write('\n\n\r\x1B[1;3;31mConnection is closed.\x1B[0m')
  dispose()
  isConnected.value = false
  showKeyboard.value = false
  emit('close', props.panelId)
}

const openTerm = () => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 16
  })

  const fitAddonInstance = new FitAddon()
  fitAddon = fitAddonInstance
  term.loadAddon(fitAddon)

  const overlayAddon = new OverlayAddon()
  term.loadAddon(overlayAddon)

  term.open(terminal.value)
  term.focus()

  disposables.push(term.onData(data => sendTermData(data)))
  disposables.push(term.onBinary(data => sendTermData(data)))

  disposables.push(term.onResize(size => {
    const msg = {type: 'winsize', cols: size.cols, rows: size.rows}
    socket.send(JSON.stringify(msg))
    overlayAddon.show(term.cols + 'x' + term.rows)
  }))

  window.addEventListener('rtty-resize', fitTerm)
  fitTerm()
  nextTick(() => term.focus())

  isConnected.value = true
}

const dispose = () => disposables.forEach(d => d.dispose())

onMounted(() => {
  const loading = ElLoading.service({
    lock: true,
    text: t('Requesting device to create terminal...'),
    background: '#555',
    customClass: 'rtty-loading'
  })

  const route = useRoute()
  const group = route.query.group ?? ''

  const protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://'

  socket = new WebSocket(protocol + location.host + `/connect/${props.devid}?group=${group}`)
  socket.binaryType = 'arraybuffer'

  socket.addEventListener('close', (ev) => {
    loading.close()

    if (ev.code === LoginErrorOffline) {
      router.push('/error/offline')
    } else if (ev.code === LoginErrorBusy) {
      router.push('/error/full')
    } else if (ev.code === LoginErrorTimeout) {
      router.push('/error/timeout')
    } else {
      closed()
    }
  })

  socket.addEventListener('error', () => {
    loading.close()

    let href = `/connect/${props.devid}`
    if (group)
      href += `?group=${group}`
    window.location.href = href
  })

  socket.addEventListener('message', ev => {
    const data = ev.data

    if (typeof data === 'string') {
      const msg = JSON.parse(data)
      if (msg.type === 'login') {
        loading.close()
        openTerm()
      } else if (msg.type === 'sendfile') {
        fileCtx.name = msg.name
        fileCtx.chunks = []
        socket.send(JSON.stringify({type: 'fileAck'}))
      } else if (msg.type === 'recvfile') {
        fileCtx.modal = true
        fileCtx.file = null
        fileCtx.accepted = false
        term.blur()
      } else if (msg.type === 'fileAck') {
        if (fileCtx.file && fileCtx.offset < fileCtx.file.size)
          readFileBlob(fileCtx.fr, fileCtx.file, fileCtx.offset, ReadFileBlkSize)
      }
    } else {
      const data = new Uint8Array(ev.data)

      if (data[0] === 0) {
        unack += data.length - 1
        term.write(data.slice(1))

        if (unack > AckBlkSize) {
          const msg = {type: 'ack', ack: unack}
          socket.send(JSON.stringify(msg))
          unack = 0
        }
      } else {
        if (data.length === 1) {
          const blob = new Blob(fileCtx.chunks)
          const url = URL.createObjectURL(blob)
          const a = document.createElement('a')
          a.href = url
          a.download = fileCtx.name
          document.body.appendChild(a)
          a.click()

          setTimeout(() => {
            fileCtx.chunks = []
            document.body.removeChild(a)
            window.URL.revokeObjectURL(url)
          }, 100)
        } else {
          fileCtx.chunks.push(data.slice(1))
          socket.send(JSON.stringify({type: 'fileAck'}))
        }
      }
    }
  })
})

onUnmounted(() => {
  window.removeEventListener('rtty-resize', fitTerm)

  dispose()

  if (term)
    term.dispose()

  if (socket)
    socket.close()
})
</script>

<style scoped>
  .terminal-container {
    height: 100%;
    position: relative;
    overflow: hidden;
  }

  .terminal {
    margin: 5px;
    height: 100%;
  }

  .floating-keyboard {
    position: absolute;
    bottom: 80px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 999;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    border-radius: 12px;
    background: rgba(248, 249, 250, 0.95);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.2);
    cursor: move;
    width: clamp(400px, 85vw, 700px);
  }

  .keyboard-toggle-btn {
    position: absolute;
    bottom: 20px;
    right: 20px;
    z-index: 1000;
    opacity: 0.8;
    transition: opacity 0.3s ease;
  }

  .keyboard-toggle-btn:hover {
    opacity: 1;
  }

  :deep(.xterm .xterm-viewport) {
    overflow-y: auto;
  }
</style>
