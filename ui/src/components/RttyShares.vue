<template>
  <el-dialog v-model="managerVisible" :title="t('Shares')" width="min(96vw, 1100px)">
    <div class="shares-toolbar">
      <el-button :icon="Refresh" :loading="loading" :title="t('Refresh List')" @click="loadShares" />
    </div>
    <div class="shares-list">
      <el-alert v-if="listError" type="error" :title="listError" show-icon class="notice" />
      <el-table v-loading="loading" :data="shares" :empty-text="t('No active shares')" class="shares-table">
        <el-table-column :label="t('Type')" width="100">
          <template #default="{ row }">{{ kindLabel(row.kind) }}</template>
        </el-table-column>
        <el-table-column :label="t('Device')" min-width="180">
          <template #default="{ row }">{{ row.group ? `${row.group} / ` : '' }}{{ row.deviceId }}</template>
        </el-table-column>
        <el-table-column :label="t('Destination')" min-width="180">
          <template #default="{ row }">
            {{ row.kind === 'tcp' ? `${row.targetIp}:${row.targetPort}` : row.kind === 'serial' ? row.serial?.Port : t('Device terminal') }}
          </template>
        </el-table-column>
        <el-table-column :label="t('Address')" min-width="180">
          <template #default="{ row }">
            <span>{{ row.host }}:{{ row.port }}</span>
            <el-button text :icon="CopyDocument" :title="t('Copy address')" @click="copy(`${row.host}:${row.port}`)" />
          </template>
        </el-table-column>
        <el-table-column :label="t('Connections')" prop="connections" width="75" />
        <el-table-column :label="t('Idle closes')" width="115">
          <template #default="{ row }">{{ row.idleDeadline ? formatDeadline(row.idleDeadline) : t('Active') }}</template>
        </el-table-column>
        <el-table-column width="70" fixed="right">
          <template #default="{ row }">
            <el-button type="danger" text :icon="Delete" :title="t('End share')" @click="endShare(row)" />
          </template>
        </el-table-column>
      </el-table>
    </div>
  </el-dialog>

  <el-dialog v-model="creating" :title="t('Create share')" width="min(94vw, 450px)">
    <el-form :model="form" @submit.prevent="createShare" label-width="auto" :rules="rules">
      <el-form-item :label="t('Device')">
        <span>{{ selectedDevice?.group ? `${selectedDevice.group} / ` : '' }}{{ selectedDevice?.id }}</span>
      </el-form-item>
      <el-form-item :label="t('Type')">
        <el-segmented v-model="form.kind" :options="kindOptions" class="full-width" />
      </el-form-item>
      <RttySerialSettings v-if="form.kind === 'serial'" ref="serialSettings"
        :dev="selectedDevice" :group="selectedDevice?.group" :active="creating"/>
      <template v-if="form.kind === 'tcp'">
        <el-form-item :label="t('Target IPv4')" prop="targetIp">
          <el-input v-model.trim="form.targetIp" />
        </el-form-item>
        <el-form-item :label="t('Target port')">
          <el-input-number v-model="form.targetPort" :min="1" :max="65535" :controls="false" class="full-width" />
        </el-form-item>
      </template>
      <el-form-item :label="t('Idle timeout')">
        <el-select v-model="form.idleSeconds">
          <el-option v-for="n in [1, 5, 15, 30, 60]" :key="n" :label="t('minutes', { count: n })" :value="n * 60" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('Public port')"><el-input-number v-model="form.port" :min="0" :max="65535" :controls="false" class="full-width" :placeholder="t('Automatic')" /></el-form-item>
      <el-alert v-if="createError" type="error" :title="createError" show-icon class="notice" />
    </el-form>
    <template #footer>
      <el-button @click="creating = false">{{ t('Cancel') }}</el-button>
      <el-button type="primary" :loading="submitting" :disabled="!canCreate" @click="createShare">{{ t('Create share') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="createdVisible" :title="t('Share created')" width="min(94vw, 520px)" @closed="created = null">
    <template v-if="created">
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="t('Address')">{{ created.share.host }}:{{ created.share.port }}</el-descriptions-item>
        <el-descriptions-item v-if="created.password" :label="t('Username')">share</el-descriptions-item>
        <el-descriptions-item v-if="created.password" :label="t('Temporary password')">{{ created.password }}</el-descriptions-item>
      </el-descriptions>
      <el-space class="created-actions">
        <el-button :icon="CopyDocument" @click="copy(created.share.kind === 'tcp' ? `${created.share.host}:${created.share.port}` : `ssh -p ${created.share.port} share@${created.share.host}`)">{{ t('Copy connection') }}</el-button>
        <el-button v-if="created.password" :icon="CopyDocument" @click="copy(created.password)">{{ t('Copy password') }}</el-button>
      </el-space>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, onUnmounted, reactive, ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CopyDocument, Delete, Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'
import RttySerialSettings from './RttySerialSettings.vue'
import useClipboard from 'vue-clipboard3'

const { t } = useI18n()
const serialSettings = useTemplateRef('serialSettings')
const shares = ref([])
const selectedDevice = ref(null)
const loading = ref(false)
const submitting = ref(false)
const listError = ref('')
const createError = ref('')
const creating = ref(false)
const managerVisible = ref(false)
const createdVisible = ref(false)
const created = ref(null)
const form = reactive({
  kind: 'terminal', port: 0, idleSeconds: 60,
  targetIp: '', targetPort: 22
})

const kindOptions = computed(() => [
  { label: t('Terminal SSH'), value: 'terminal' },
  { label: t('Serial SSH'), value: 'serial', disabled: selectedDevice.value?.proto < 6 },
  { label: t('TCP port'), value: 'tcp', disabled: selectedDevice.value?.proto < 6 }
])

const validIp = computed(() => {
  const parts = form.targetIp.split('.')
  return parts.length === 4 && parts.every(part => /^\d{1,3}$/.test(part) &&
    (part.length === 1 || part[0] !== '0') && Number(part) <= 255)
})

const canCreate = computed(() => selectedDevice.value && !submitting.value &&
  (form.kind === 'terminal' || (selectedDevice.value.proto >= 6 &&
    (form.kind === 'serial' ? !!serialSettings.value?.ready : validIp.value && form.targetPort >= 1))))
let refreshTimer

const kindLabel = kind => ({ terminal: t('Terminal SSH'), serial: t('Serial SSH'), tcp: t('TCP port') })[kind] || kind
const formatDeadline = value => new Date(value).toLocaleString()

const { toClipboard } = useClipboard()

const copyText = async(text) => {
  try {
    await toClipboard(text)
    return Promise.resolve()
  } catch (err) {
    return Promise.reject(err)
  }
}

const copy = async value => {
  copyText(value).then(() => ElMessage.success(t('Copied to clipboard')))
}

const loadShares = async() => {
  loading.value = true
  try {
    shares.value = (await axios.get('/api/shares')).data
    listError.value = ''
  } catch {
    listError.value = t('Unable to load shares')
  } finally {
    loading.value = false
  }
}

watch(() => form.kind, () => {
  createError.value = ''
})

const rules = {
  targetIp: [
    { required: true, message: t('Please enter a target IP'), trigger: 'blur' },
    { validator: (rule, value, callback) => {
      const parts = value.split('.')
      if (parts.length !== 4 || !parts.every(part => /^\d{1,3}$/.test(part) &&
        (part.length === 1 || part[0] !== '0') && Number(part) <= 255)) {
        callback(new Error(t('Please enter a valid IPv4 address')))
      } else {
        callback()
      }
    }, trigger: 'blur' }
  ]
}

const openCreate = (dev, group = '') => {
  selectedDevice.value = dev ? { ...dev, group } : null
  form.kind = 'terminal'
  form.port = 0
  form.idleSeconds = 60
  createError.value = ''
  managerVisible.value = false
  creating.value = true
}

const openManager = () => {
  managerVisible.value = true
  loadShares()
}

defineExpose({ openCreate, openManager })
const createShare = async() => {
  if (!canCreate.value) return

  const serial = form.kind === 'serial' ? serialSettings.value.getSettings() : null
  if (form.kind === 'serial' && !serial) return

  const dev = selectedDevice.value
  submitting.value = true
  createError.value = ''
  try {
    const body = { kind: form.kind, group: dev.group, deviceId: dev.id,
      idleSeconds: form.idleSeconds, port: form.port || 0 }
    if (form.kind === 'tcp') Object.assign(body, { targetIp: form.targetIp, targetPort: form.targetPort })
    if (serial) body.serial = {
      ...serial, parity: { none: 0, odd: 1, even: 2 }[serial.parity]
    }
    created.value = (await axios.post('/api/shares', body)).data
    creating.value = false
    createdVisible.value = true
    await loadShares()
  } catch (err) {
    createError.value = err.response?.data?.error || t('Unable to create share')
  } finally {
    submitting.value = false
  }
}

const endShare = async row => {
  try {
    await ElMessageBox.confirm(t('End share confirmation'), t('End share'), { type: 'warning' })
    await axios.delete(`/api/shares/${row.id}`)
    await loadShares()
  } catch (err) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(t('Unable to end share'))
  }
}

watch(managerVisible, visible => {
  clearInterval(refreshTimer)
  if (visible) refreshTimer = setInterval(loadShares, 10000)
})
onUnmounted(() => clearInterval(refreshTimer))
</script>

<style scoped>
.shares-toolbar { display: flex; justify-content: flex-end; gap: 8px; margin-bottom: 12px; }
.shares-list { overflow-x: auto; }
.shares-table { width: 100%; }
.full-width { width: 100%; }
.target-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 16px; }
.notice { margin: 8px 0 16px; }
.created-actions { margin-top: 18px; flex-wrap: wrap; }
@media (max-width: 600px) {
  .shares-header { padding: 0 12px; }
  .target-grid { grid-template-columns: 1fr; }
}
</style>
