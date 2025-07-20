<template>
  <el-dialog v-model="model" :title="$t('Access your devices\'s Web')" width="300">
    <el-form ref="form" :label-width="80" label-position="left" :model="formData" :rules="ruleValidate">
      <el-form-item :label="$t('Proto')" prop="proto">
        <el-radio-group v-model="formData.proto">
          <el-radio value="http" size="large">HTTP</el-radio>
          <el-radio value="https" size="large">HTTPS</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="$t('ipaddr')" prop="ipaddr">
        <el-input v-model="formData.ipaddr" placeholder="127.0.0.1"/>
      </el-form-item>
      <el-form-item :label="$t('port')" prop="port">
        <el-input v-model.number="formData.port" :placeholder="formData.proto === 'https' ? '443' : '80'"/>
      </el-form-item>
      <el-form-item :label="$t('path')" prop="path">
        <el-input v-model="formData.path" placeholder="/"/>
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="model = false">{{ $t('Cancel') }}</el-button>
        <el-button type="primary" @click="open">{{ $t('OK') }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { reactive, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'

const props = defineProps({
  dev: Object
})

const { t } = useI18n()

const form = useTemplateRef('form')
const model = defineModel()

const formData = reactive({
  proto: 'http',
  ipaddr: '',
  port: null,
  path: ''
})

const isValidIP = (addr) => {
  const ipv4Pattern = '(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)'
  return new RegExp(ipv4Pattern).test(addr)
}

const ruleValidate = {
  ipaddr: [{validator: (rule, value, callback) => {
    if (!value) {
      callback()
      return
    }

    if (!isValidIP(value)) {
      callback(new Error(t('Invalid IP address')))
    }

    callback()
  }}],
  port: [{validator: (rule, value, callback) => {
    if (!value) {
      callback()
      return
    }

    if (!Number.isInteger(value) || value < 1 || value > 65536) {
      callback(new Error(t('Invalid port')))
    }

    callback()
  }}],
  path: [{validator: (rule, value, callback) => {
    if (!value) {
      callback()
      return
    }

    if (!value.startsWith('/')) {
      callback(new Error(t('Must start with /')))
    }

    callback()
  }}]
}

const open = () => {
  form.value.validate(valid => {
    if (!valid)
      return

    const dev = props.dev

    if (dev.proto < 4 && formData.proto === 'https') {
      ElMessage.error(t('Your device\'s rtty does not support https proxy, please upgrade it.'))
      return
    }

    model.value = false

    setTimeout(() => {
      const proto = formData.proto
      let ipaddr = formData.ipaddr
      let port = formData.port
      let path = formData.path

      if (!ipaddr)
        ipaddr = '127.0.0.1'

      if (!port)
        port = proto === 'https' ? 443 : 80

      if (!path)
        path = '/'

      const addr = encodeURIComponent(`${ipaddr}:${port}${path}`)

      const group = dev.group
      const devid = dev.id

      if (group)
        window.open(`/web2/${group}/${devid}/${proto}/${addr}`)
      else
        window.open(`/web/${devid}/${proto}/${addr}`)
    }, 100)
  })
}
</script>
