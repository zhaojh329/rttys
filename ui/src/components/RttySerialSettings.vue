<template>
  <el-form-item :label="t('Serial port')">
    <el-space>
      <el-select v-model="settings.port" filterable :loading="loading" :placeholder="t('Select serial port')">
        <el-option v-for="port in ports" :key="port" :label="port" :value="port"/>
      </el-select>
      <el-button :icon="Refresh" :loading="loading" :title="t('Refresh List')" @click="loadPorts"/>
    </el-space>
  </el-form-item>
  <el-alert v-if="error" :title="error" type="error" :closable="false" class="list-error"/>
  <el-form-item :label="t('Baud rate')">
    <el-select v-model="settings.baudRate" filterable default-first-option>
      <el-option v-for="rate in baudRates" :key="rate" :label="rate" :value="rate"/>
    </el-select>
  </el-form-item>
  <el-form-item :label="t('Data bits')">
    <el-select v-model="settings.dataBits">
      <el-option v-for="bits in [5, 6, 7, 8]" :key="bits" :label="bits" :value="bits"/>
    </el-select>
  </el-form-item>
  <el-form-item :label="t('Stop bits')">
    <el-select v-model="settings.stopBits">
      <el-option v-for="bits in [1, 2]" :key="bits" :label="bits" :value="bits"/>
    </el-select>
  </el-form-item>
  <el-form-item :label="t('Parity')">
    <el-select v-model="settings.parity">
      <el-option :label="t('None')" value="none"/>
      <el-option :label="t('Odd')" value="odd"/>
      <el-option :label="t('Even')" value="even"/>
    </el-select>
  </el-form-item>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh } from '@element-plus/icons-vue'
import axios from 'axios'

const props = defineProps({ dev: Object, group: String, active: Boolean })
const { t } = useI18n()
const ports = ref([])
const loading = ref(false)
const error = ref('')
const baudError = ref('')
const baudRates = ['9600', '19200', '38400', '57600', '115200', '230400', '460800', '921600']
const settings = reactive({ port: '', baudRate: '115200', dataBits: 8, stopBits: 1, parity: 'none' })
const ready = computed(() => props.active && !!props.dev && !loading.value && !error.value && !!settings.port)
let latestRequest = 0

const loadPorts = () => {
  const request = ++latestRequest
  settings.port = ''
  error.value = ''
  baudError.value = ''
  ports.value = []
  loading.value = false

  if (!props.active || !props.dev) return

  loading.value = true
  axios.get(`/api/serial-ports/${encodeURIComponent(props.dev.id)}`, {
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

const getSettings = () => {
  const currentSettings = { ...settings }
  currentSettings.baudRate = parseInt(currentSettings.baudRate, 10)
  return currentSettings
}

watch(() => [props.active, props.dev?.id, props.group], loadPorts, { immediate: true })
defineExpose({ ready, getSettings })
</script>

<style scoped>
.list-error { margin-top: 8px; }
</style>
