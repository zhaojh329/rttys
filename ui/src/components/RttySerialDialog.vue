<template>
  <el-dialog v-model="model" :title="$t('Open serial port')" width="320" @open="loadPorts">
    <el-form :model="settings" @submit.prevent="openSerialPort" label-width="auto">
      <el-form-item :label="$t('Serial port')" prop="port">
        <el-space>
          <el-select v-model="settings.port" filterable :loading="loading" :placeholder="$t('Select serial port')">
            <el-option v-for="port in ports" :key="port" :label="port" :value="port"/>
          </el-select>
          <el-button :icon="Refresh" :loading="loading" :title="$t('Refresh List')" @click="loadPorts"/>
        </el-space>
      </el-form-item>
      <el-alert v-if="error" :title="error" type="error" :closable="false" class="list-error"/>
      <el-text v-else-if="!loading && ports.length === 0" type="info">{{ $t('No serial ports found') }}</el-text>
      <el-form-item :label="$t('Baud rate')" prop="baudRate" :error="baudError">
        <el-select v-model="settings.baudRate" filterable allow-create default-first-option @change="baudError = ''">
          <el-option v-for="rate in baudRates" :key="rate" :label="rate" :value="rate"/>
        </el-select>
      </el-form-item>
      <el-form-item :label="$t('Data bits')" prop="dataBits">
        <el-select v-model="settings.dataBits">
          <el-option v-for="bits in [5, 6, 7, 8]" :key="bits" :label="bits" :value="bits"/>
        </el-select>
      </el-form-item>
      <el-form-item :label="$t('Stop bits')" prop="stopBits">
        <el-select v-model="settings.stopBits">
          <el-option v-for="bits in [1, 2]" :key="bits" :label="bits" :value="bits"/>
        </el-select>
      </el-form-item>
      <el-form-item :label="$t('Parity')" prop="parity">
        <el-select v-model="settings.parity">
          <el-option :label="$t('None')" value="none"/>
          <el-option :label="$t('Odd')" value="odd"/>
          <el-option :label="$t('Even')" value="even"/>
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="model = false">{{ $t('Cancel') }}</el-button>
      <el-button type="primary" :disabled="loading || !settings.port || !!error" @click="openSerialPort">{{ $t('Open') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh } from '@element-plus/icons-vue'
import axios from 'axios'

const props = defineProps({ dev: Object, group: String })
const model = defineModel()
const { t } = useI18n()
const ports = ref([])
const loading = ref(false)
const error = ref('')
const baudError = ref('')
const baudRates = ['9600', '19200', '38400', '57600', '115200', '230400', '460800', '921600']
const settings = reactive({ port: '', baudRate: '115200', dataBits: 8, stopBits: 1, parity: 'none' })
let latestRequest = 0

const loadPorts = () => {
  if (!props.dev) return

  const request = ++latestRequest
  loading.value = true
  settings.port = ''
  error.value = ''
  baudError.value = ''
  ports.value = []

  axios.get(`/serial-ports/${props.dev.id}`, {
    params: { group: props.group || '' }
  }).then(res => {
    if (request !== latestRequest) return
    ports.value = res.data.ports
  }).catch(() => {
    if (request !== latestRequest) return
    error.value = t('Unable to list serial ports')
  }).finally(() => {
    if (request !== latestRequest) return
    loading.value = false
  })
}

const openSerialPort = () => {
  if (!props.dev || loading.value || error.value || !settings.port) return

  const baudRate = String(settings.baudRate)
  const rate = Number(baudRate)
  if (!/^\d+$/.test(baudRate) || !Number.isInteger(rate) || rate < 300 || rate > 4000000) {
    baudError.value = t('Invalid baud rate')
    return
  }

  baudError.value = ''

  model.value = false

  setTimeout(() => {
    const params = new URLSearchParams({
      group: props.group || '', port: settings.port, baudRate: String(rate),
      dataBits: String(settings.dataBits), stopBits: String(settings.stopBits), parity: settings.parity
    })
    window.open(`/serial/${props.dev.id}?${params}`, '_blank')
  }, 100)
}
</script>

<style scoped>
.list-error { margin-top: 8px; }
</style>
