<template>
  <div class="terminal-container">
    <div ref="terminal" class="rtty-terminal" @contextmenu.prevent="showContextmenu"></div>
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
    <transition name="find-box">
      <div v-if="showFindBox" class="find-box" @keydown="onFindBoxKeydown">
        <el-input ref="findInput" class="find-input" v-model="findText" :placeholder="$t('Find')" clearable @keydown.enter="doFind('next')"/>
        <el-checkbox-group class="find-flags" v-model="findFlags">
          <el-checkbox value="caseSensitive" :label="$t('Match Case')"/>
          <el-checkbox value="wholeWord" :label="$t('Whole Word')"/>
          <el-checkbox value="regex" :label="$t('Regular')"/>
        </el-checkbox-group>
        <el-button size="small" @click="doFind('prev')">{{ $t('Prev') }}</el-button>
        <el-button size="small" @click="doFind('next')">{{ $t('Next') }}</el-button>
        <el-button size="small" @click="showFindBox = false">{{ $t('Close') }}</el-button>
      </div>
    </transition>
    <ContextMenu v-model="contextmenuPos" :menus="contextmenus" @click="onContextmenuClick"/>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick, useTemplateRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElLoading, ElMessage, ElMessageBox } from 'element-plus'
import useClipboard from 'vue-clipboard3'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { SearchAddon } from '@xterm/addon-search'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import OverlayAddon from '../xterm-addon/xterm-addon-overlay'
import ContextMenu from '../components/ContextMenu.vue'
import RttyKeyboard from '../components/RttyKeyboard.vue'

const LoginErrorOffline = 4000
const LoginErrorBusy = 4001
const LoginErrorTimeout = 4002
const PeerCapabilitySignal = 1
const PeerSignalOffer = 0
const PeerSignalAnswer = 1
const PeerSignalCandidate = 2
const PeerSignalReady = 3
const PeerSignalFailed = 4
const PeerSignalFallbackRelay = 5

const MsgTypeFileData = 0x03

const ReadFileBlkSize = 63 * 1024

const AckBlkSize = 4 * 1024

const props = defineProps({
  devid: String,
  panelId: String
})

const emit = defineEmits(['split', 'close'])

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const { toClipboard } = useClipboard()

const terminal = useTemplateRef('terminal')
const contextmenuPos = ref(null)

const contextmenus = [
  {name: 'copy', caption: t('Copy'), shortcut: 'Ctrl + Insert'},
  {name: 'paste', caption: t('Paste'), shortcut: 'Shift + Insert'},
  {name: 'clear', caption: t('Clear Scrollback')},
  {name: 'find', caption: t('Find'), shortcut: 'Ctrl + F'},
  {name: 'clear-highlighting', caption: t('Clear Highlighting')},
  {name: 'font+', caption: t('font+'), shortcut: 'Ctrl + ↑'},
  {name: 'font-', caption: t('font-'), shortcut: 'Ctrl + ↓'},
  {name: 'upload', caption: t('Upload file'), shortcut: 'rtty -R'},
  {name: 'download', caption: t('Download file'), shortcut: 'rtty -S file'},
  {name: 'split-left', caption: t('split-left')},
  {name: 'split-right', caption: t('split-right')},
  {name: 'split-up', caption: t('split-up')},
  {name: 'split-down', caption: t('split-down')},
  {name: 'close', caption: t('Close')},
  {name: 'about', caption: t('About')}
]

const showFindBox = ref(false)
const findInput = useTemplateRef('findInput')
const findText = ref('')
const findFlags = ref([])

const openFindBox = () => {
  showFindBox.value = true
  nextTick(() => findInput.value.focus())
}

const onFindBoxKeydown = (e) => {
  if (e.key === 'Escape') {
    showFindBox.value = false
  }
}

watch(() => showFindBox.value, (val) => {
  if (!val) {
    nextTick(() => term.focus())
  }
})

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
let searchAddon = null
let unack = 0
let peerConnection = null
let peerDataChannel = null
let peerIceServers = []
let peerFallbackTimer = null
const showKeyboard = ref(false)
const isConnected = ref(false)
const peerTransportRequested = ref(false)
const peerTransportMode = ref('server-relay')
const peerProbeSent = ref(false)
const lastPeerNotice = ref('')

const routeQueryValue = (value) => Array.isArray(value) ? value[0] : (value ?? '')

const wantsPeerTransport = () => {
  const transport = routeQueryValue(route.query.transport)
  const peer = routeQueryValue(route.query.peer)

  return transport === 'peer' || peer === '1' || peer === 'true'
}

const buildConnectPath = () => {
  const params = new URLSearchParams()
  const group = routeQueryValue(route.query.group)

  if (group)
    params.set('group', group)

  if (peerTransportRequested.value)
    params.set('transport', 'peer')

  const query = params.toString()

  return `/connect/${props.devid}${query ? `?${query}` : ''}`
}

const buildWebSocketUrl = () => {
  const protocol = location.protocol === 'https:' ? 'wss://' : 'ws://'
  return protocol + location.host + buildConnectPath()
}

const buildDeviceInfoPath = () => {
  const params = new URLSearchParams()
  const group = routeQueryValue(route.query.group)

  if (group)
    params.set('group', group)

  const query = params.toString()
  return `/dev/${props.devid}${query ? `?${query}` : ''}`
}

const loadPeerConfig = async() => {
  if (!peerTransportRequested.value)
    return

  try {
    const resp = await fetch(buildDeviceInfoPath(), { credentials: 'same-origin' })
    if (!resp.ok)
      return

    const dev = await resp.json()
    peerIceServers = Array.isArray(dev.peerIceServers) ? dev.peerIceServers : []

    if ((dev.capabilities & PeerCapabilitySignal) === 0) {
      peerTransportRequested.value = false
      peerTransportMode.value = 'server-relay'
    }
  } catch (error) {
    console.error('load peer config failed', error)
  }
}

const sendPeerSignal = (signalType, payload) => {
  if (!socket)
    return

  socket.send(JSON.stringify({type: 'peerSignal', signalType, payload}))
}

const sendPeerControl = (payload) => {
  if (peerTransportMode.value !== 'peer' || !peerDataChannel || peerDataChannel.readyState !== 'open')
    return false

  peerDataChannel.send(JSON.stringify(payload))
  return true
}

const showPeerNotice = (text, level = 'info') => {
  if (lastPeerNotice.value === text)
    return

  lastPeerNotice.value = text

  if (level === 'success')
    ElMessage.success(text)
  else if (level === 'warning')
    ElMessage.warning(text)
  else
    ElMessage.info(text)

  if (term)
    term.writeln(`\r\n[peer] ${text}`)
}

const clearPeerFallbackTimer = () => {
  if (peerFallbackTimer !== null) {
    clearTimeout(peerFallbackTimer)
    peerFallbackTimer = null
  }
}

const closePeerConnection = () => {
  clearPeerFallbackTimer()

  if (peerDataChannel) {
    peerDataChannel.close()
    peerDataChannel = null
  }

  if (peerConnection) {
    peerConnection.onicecandidate = null
    peerConnection.onconnectionstatechange = null
    peerConnection.ondatachannel = null
    peerConnection.close()
    peerConnection = null
  }
}

const supportsPeerTransport = () => typeof RTCPeerConnection !== 'undefined'

const setupPeerDataChannel = (dc) => {
  peerDataChannel = dc

  dc.onopen = () => {
    peerTransportMode.value = 'negotiating'
  }

  dc.onclose = () => {
    peerDataChannel = null
  }

  dc.onmessage = (event) => {
    const data = typeof event.data === 'string'
      ? new TextEncoder().encode(event.data)
      : new Uint8Array(event.data)

    term.write(data)
  }
}

const requestPeerTransport = async() => {
  if (!peerTransportRequested.value || peerProbeSent.value || !socket)
    return

  if (!supportsPeerTransport()) {
    peerTransportRequested.value = false
    peerTransportMode.value = 'server-relay'
    const text = 'WebRTC is not supported by this browser, using server relay.'
    ElMessage.info(text)
    if (term)
      term.writeln(`\r\n[peer] ${text}`)
    return
  }

  peerProbeSent.value = true
  peerTransportMode.value = 'negotiating'
  closePeerConnection()
  showPeerNotice('Attempting experimental peer transport...')

  try {
    peerConnection = new RTCPeerConnection({ iceServers: peerIceServers })
    peerConnection.onicecandidate = (event) => {
      if (event.candidate)
        sendPeerSignal(PeerSignalCandidate, event.candidate.toJSON())
    }

    peerConnection.onconnectionstatechange = () => {
      if (!peerConnection)
        return

      const state = peerConnection.connectionState
      if (state === 'failed' || state === 'disconnected') {
        peerTransportMode.value = 'server-relay'
        clearPeerFallbackTimer()
        peerFallbackTimer = setTimeout(() => {
          peerFallbackTimer = null
          showPeerNotice(`Peer connection ${state}, using server relay.`)
        }, 800)
        closePeerConnection()
      }
    }

    const dc = peerConnection.createDataChannel('terminal')
    setupPeerDataChannel(dc)

    const offer = await peerConnection.createOffer()
    await peerConnection.setLocalDescription(offer)

    sendPeerSignal(PeerSignalOffer, {
      offer: peerConnection.localDescription ?? offer,
      iceServers: peerIceServers
    })
  } catch (error) {
    peerTransportMode.value = 'server-relay'
    showPeerNotice(`Peer negotiation failed, using server relay: ${error.message ?? error}`)
    closePeerConnection()
  }
}

const peerMessageText = (prefix, reason = '') => {
  if (!reason)
    return prefix

  return `${prefix}: ${reason}`
}

const handlePeerSignal = async(msg) => {
  if (msg.signalType === PeerSignalAnswer) {
    if (!peerConnection || !msg.payload)
      return

    await peerConnection.setRemoteDescription(msg.payload)
    return
  }

  if (msg.signalType === PeerSignalCandidate) {
    if (!peerConnection || !msg.payload)
      return

    await peerConnection.addIceCandidate(msg.payload)
    return
  }

  if (msg.signalType === PeerSignalReady) {
    clearPeerFallbackTimer()
    peerTransportMode.value = 'peer'
    showPeerNotice('Experimental peer transport is ready.', 'success')
    return
  }

  if (msg.signalType === PeerSignalFailed) {
    clearPeerFallbackTimer()
    peerTransportMode.value = 'server-relay'
    const reason = typeof msg.payload === 'object' && msg.payload !== null ? msg.payload.reason : ''
    showPeerNotice(peerMessageText('Experimental peer transport failed, using server relay', reason))
    closePeerConnection()
  }

  if (msg.signalType === PeerSignalFallbackRelay) {
    clearPeerFallbackTimer()
    peerTransportMode.value = 'server-relay'

    const reason = typeof msg.payload === 'object' && msg.payload !== null ? msg.payload.reason : ''
    showPeerNotice(peerMessageText('Experimental peer transport unavailable, using server relay', reason))
  }
}

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
  } else if (name === 'find') {
    openFindBox()
    return
  } else if (name === 'clear-highlighting') {
    searchAddon.clearDecorations()
    term.clearSelection()
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

const doFind = (type) => {
  const options = {
    decorations: {
      matchBackground: '#2e7d32',
      matchBorder: '#2e7d32',
      matchOverviewRuler: '#2e7d32',
      activeMatchBackground: '#ff8f00',
      activeMatchBorder: '#ff8f00',
      activeMatchColorOverviewRuler: '#ff8f00'
    }
  }

  findFlags.value.forEach(v => {
    options[v] = true
  })

  if (type === 'next')
    searchAddon.findNext(findText.value, options)
  else
    searchAddon.findPrevious(findText.value, options)
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

const sendTermData = (data) => {
  const encoded = new TextEncoder().encode(data)

  if (peerTransportMode.value === 'peer' && peerDataChannel && peerDataChannel.readyState === 'open') {
    peerDataChannel.send(encoded)
    return
  }

  socket.send(new Uint8Array([0, ...encoded]))
}

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
  peerTransportMode.value = 'server-relay'
  peerProbeSent.value = false
  lastPeerNotice.value = ''
  closePeerConnection()
}

const openTerm = () => {
  term = new Terminal({
    allowProposedApi: true,
    cursorBlink: true,
    cursorStyle: 'bar',
    cursorInactiveStyle: 'none',
    fontSize: 16
  })

  term.loadAddon(new WebLinksAddon())

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  searchAddon = new SearchAddon()
  term.loadAddon(searchAddon)

  const overlayAddon = new OverlayAddon()
  term.loadAddon(overlayAddon)

  term.open(terminal.value)
  term.focus()

  disposables.push(term.onData(data => sendTermData(data)))
  disposables.push(term.onBinary(data => sendTermData(data)))

  disposables.push(term.onResize(size => {
    const msg = {type: 'winsize', cols: size.cols, rows: size.rows}
    if (!sendPeerControl(msg))
      socket.send(JSON.stringify(msg))
    overlayAddon.show(term.cols + 'x' + term.rows)
  }))

  disposables.push(term.onKey(({ domEvent }) => {
    const e = domEvent
    if (e.ctrlKey || e.metaKey) {
      const key = e.key.toLowerCase()
      if (key === 'f')
        openFindBox()
      else if (key === 'arrowup')
        updateFontSize(1)
      else if (key === 'arrowdown')
        updateFontSize(-1)
    }
  }))

  window.addEventListener('rtty-resize', fitTerm)
  fitTerm()
  nextTick(() => term.focus())

  isConnected.value = true

  term.writeln([
    ' ┌─────────────────────────────────────────────────────────────┐',
    ' │  \x1b[36;1mPowered by rtty\x1b[0m(https://github.com/zhaojh329/rtty)         │',
    ' │                                                             │',
    ' │ ⮞ For more operations, please right-click the mouse.        │',
    ' └─────────────────────────────────────────────────────────────┘',
    ''
  ].join('\n\r'))
}

const dispose = () => disposables.forEach(d => d.dispose())

onMounted(() => {
  peerTransportRequested.value = wantsPeerTransport()
  loadPeerConfig().finally(() => {
  const loading = ElLoading.service({
    lock: true,
    text: t('Requesting device to create terminal...'),
    background: '#555',
    customClass: 'rtty-loading'
  })

  socket = new WebSocket(buildWebSocketUrl())
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
    window.location.href = buildConnectPath()
  })

    socket.addEventListener('message', async(ev) => {
    const data = ev.data

    if (typeof data === 'string') {
      const msg = JSON.parse(data)
      if (msg.type === 'login') {
        loading.close()
        openTerm()
        if (peerTransportRequested.value)
          requestPeerTransport()
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
      } else if (msg.type === 'peerSignal') {
        await handlePeerSignal(msg)
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
})

onUnmounted(() => {
  window.removeEventListener('rtty-resize', fitTerm)

  dispose()

  if (term)
    term.dispose()

  if (socket)
    socket.close()

  closePeerConnection()
})
</script>

<style scoped>
  .terminal-container {
    height: 100%;
    position: relative;
    overflow: hidden;
  }

  .rtty-terminal {
    height: 100%;
    background-color: black;
  }

  :deep(.terminal) {
    padding: 5px;
  }

  :deep(.xterm .xterm-viewport) {
    overflow-y: auto;
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

  .find-box {
    width: auto;
    position: absolute;
    top: 5px;
    right: 15px;
    z-index: 1001;
    background: rgba(255,255,255,0.98);
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.15);
    padding: 4px 4px;
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .find-box-enter-active,
  .find-box-leave-active {
    transition: all 0.3s cubic-bezier(.55,0,.1,1);
  }
  .find-box-enter-from,
  .find-box-leave-to {
    opacity: 0;
    transform: translateY(-40px);
  }
  .find-box-enter-to,
  .find-box-leave-from {
    opacity: 1;
    transform: translateY(0);
  }

  .find-input {
    width: 200px;
  }

  .find-flags {
    display: flex;
    gap: 5px
  }

  :deep(.el-checkbox) {
    margin-right: 1px;
  }
</style>
