<template>
  <el-dialog v-model="model" :title="$t('Open serial port')" width="320">
    <el-form @submit.prevent="openSerialPort" label-width="auto">
      <RttySerialSettings ref="serialSettings" :dev="dev" :group="group" :active="model"/>
    </el-form>
    <template #footer>
      <el-button @click="model = false">{{ $t('Cancel') }}</el-button>
      <el-button type="primary" :disabled="!serialSettings?.ready" @click="openSerialPort">{{ $t('Open') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { useTemplateRef } from 'vue'
import RttySerialSettings from './RttySerialSettings.vue'

const props = defineProps({ dev: Object, group: String })
const model = defineModel()
const serialSettings = useTemplateRef('serialSettings')

const openSerialPort = () => {
  if (!props.dev) return

  const settings = serialSettings.value.getSettings()

  model.value = false

  setTimeout(() => {
    const params = new URLSearchParams({
      group: props.group || '', port: settings.port, baudRate: String(settings.baudRate),
      dataBits: String(settings.dataBits), stopBits: String(settings.stopBits), parity: settings.parity
    })
    window.open(`/serial/${props.dev.id}?${params}`, '_blank')
  }, 100)
}
</script>
