<template>
  <el-dialog v-model="modal" :title="$t('Access your devices\'s Web')" width="300">
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
        <el-button @click="modal = false">{{ $t('Cancel') }}</el-button>
        <el-button type="primary" @click="open">{{ $t('OK') }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
export default {
  name: 'RttyWeb',
  data() {
    return {
      modal: false,
      formData: {
        proto: 'http',
        ipaddr: '',
        port: null,
        path: ''
      },
      ruleValidate: {
        ipaddr: [{validator: (rule, value, callback) => {
          if (!value) {
            callback()
            return
          }

          if (!this.isValidIP(value)) {
            callback(new Error(this.$t('Invalid IP address')))
          }

          callback()
        }}],
        port: [{validator: (rule, value, callback) => {
          if (!value) {
            callback()
            return
          }

          if (!Number.isInteger(value) || value < 1 || value > 65536) {
            callback(new Error(this.$t('Invalid port')))
          }

          callback()
        }}],
        path: [{validator: (rule, value, callback) => {
          if (!value) {
            callback()
            return
          }

          if (!value.startsWith('/')) {
            callback(new Error(this.$t('Must start with /')))
          }

          callback()
        }}]
      },
      group: '',
      devid: '',
      devProto: null
    }
  },
  methods: {
    show(dev) {
      this.group = dev.group
      this.devid = dev.id
      this.devProto = dev.proto
      this.formData.proto = 'http'
      this.formData.ipaddr = ''
      this.formData.port = null
      this.formData.path = ''
      this.modal = true
    },
    isValidIP(addr) {
      const ipv4Pattern = '(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)'
      return new RegExp(ipv4Pattern).test(addr)
    },
    open() {
      this.$refs.form.validate(valid => {
        if (!valid)
          return

        if (this.devProto < 4 && this.formData.proto === 'https') {
          this.$message.error(this.$t('Your device\'s rtty does not support https proxy, please upgrade it.'))
          return
        }

        this.modal = false

        setTimeout(() => {
          const proto = this.formData.proto
          let ipaddr = this.formData.ipaddr
          let port = this.formData.port
          let path = this.formData.path

          if (!ipaddr)
            ipaddr = '127.0.0.1'

          if (!port)
            port = proto === 'https' ? 443 : 80

          if (!path)
            path = '/'

          const addr = encodeURIComponent(`${ipaddr}:${port}${path}`)

          if (this.group)
            window.open(`/web2/${this.group}/${this.devid}/${proto}/${addr}`)
          else
            window.open(`/web/${this.devid}/${proto}/${addr}`)
        }, 100)
      })
    }
  }
}
</script>
