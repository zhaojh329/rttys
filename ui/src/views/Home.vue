<template>
  <div style="padding:5px;">
    <div style="display: flex; justify-content: space-between;">
      <el-space>
        <el-button type="primary" round icon="Refresh" @click="handleRefresh" :disabled="loading">{{ $t('Refresh List') }}</el-button>
        <el-input style="width:200px" v-model="filterString" search @input="handleSearch" :placeholder="$t('Please enter the filter key...')"/>
        <el-button @click="showCmdForm" type="primary" :disabled="cmdStatus.execing > 0">{{ $t('Execute command') }}</el-button>
      </el-space>
      <el-space style="float: right;">
        <span style="color: var(--el-color-primary); font-size: 24px">{{ $t('device-count', {count: devlists.length}) }}</span>
        <el-divider direction="vertical" />
        <el-button type="primary" @click="handleLogout">{{ $t('Sign out') }}</el-button>
      </el-space>
    </div>
    <el-table :loading="loading" :data="pagedevlists" style="margin-top: 10px; margin-bottom: 10px; width: 100%" :empty-text="$t('No devices connected')" @selection-change='handleSelection'>
      <el-table-column type="selection" width="60" />
      <el-table-column prop="id" :label="$t('Device ID')" width="200" />
      <el-table-column :label="$t('Connected time')" width="200">
        <template #default="{ row }">
          <span>{{ formatTime(row.connected) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('Uptime')" width="200">
        <template #default="{ row }">
          <span>{{ formatTime(row.uptime) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="description" :label="$t('Description')" show-overflow-tooltip />
      <el-table-column width="200">
        <template #default="{ row }">
          <el-space>
            <el-tooltip placement="top" :content="$t('Access your device\'s Shell')">
              <el-icon size="25" color="black" style="cursor:pointer;" @click="connectDevice(row.id)"><TerminalIcon /></el-icon>
            </el-tooltip>
            <el-tooltip placement="top" :content="$t('Access your devices\'s Web')">
              <el-icon size="25" color="#409EFF" style="cursor:pointer;" @click="connectDeviceWeb(row)"><IEIcon /></el-icon>
            </el-tooltip>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination background layout="prev, pager, next, total,sizes" :total="filteredDevices.length" @change="handlePageChange"/>
    <el-dialog v-model="cmdModal" :title="$t('Execute command')" width="500">
      <el-form ref="cmdForm" :model="cmdData" :rules="cmdRuleValidate" :label-width="100" label-position="left">
        <el-form-item :label="$t('Username')" prop="username">
          <el-input v-model="cmdData.username"/>
        </el-form-item>
        <el-form-item :label="$t('Command')" prop="cmd">
          <el-input v-model.trim="cmdData.cmd"/>
        </el-form-item>
        <el-form-item :label="$t('Parameter')" prop="params">
          <el-tag v-for="tag in cmdData.params" :key="tag" closable @close="delCmdParam(tag)">{{ tag }}</el-tag>
          <el-input v-if="inputParamVisible" style="width: 90px; margin-left: 10px;" v-model="inputParamValue"
                ref="inputParam" size="small" @keyup.enter="handleInputParamConfirm" @blur="handleInputParamConfirm"/>
          <el-button v-else style="width: 40px; margin-left: 10px;" size="small" icon="plus" type="primary" @click="showInputParam"/>
        </el-form-item>
        <el-form-item :label="$t('Wait Time')" prop="wait">
          <el-input v-model.number="cmdData.wait" placeholder="30">
            <template #append>s</template>
          </el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cmdModal = false">{{ $t('Cancel') }}</el-button>
        <el-button type="primary" @click="doCmd">{{ $t('OK') }}</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="cmdStatus.modal" :title="$t('Status of executive command')" :width="800"
      :show-close="false" :close-on-press-escape="false" :close-on-click-modal="false">
      <el-progress text-inside :stroke-width="20" :percentage="cmdStatusPercent"/>
      <p>{{ $t('cmd-status-total', {count: cmdStatus.total}) }}</p>
      <p>{{ $t('cmd-status-fail', {count: cmdStatus.fail}) }}</p>
      <template #footer>
        <el-button type="primary" :disabled="cmdStatus.execing > 0" @click="showCmdResp">{{ $t('OK') }}</el-button>
        <el-button type="error" :disabled="cmdStatus.execing === 0" @click="ignoreCmdResp">{{ $t('Ignore') }}</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="cmdStatus.respModal" :title="$t('Response of executive command')" :width="800">
      <el-table :data="cmdStatus.responses" :empty-text="$t('No Response')">
        <el-table-column type="expand" width="50">
          <template #default="{ row }">
            <p>{{ $t('Stdout') + ':' }}</p>
            <el-input type="textarea" :value="row.stdout" readonly :autosize="{minRows: 1, maxRows: 20}"/>
            <p style="margin-top: 10px">{{ $t('Stderr') + ':' }}</p>
            <el-input type="textarea" :value="row.stderr" readonly :autosize="{minRows: 1, maxRows: 20}"/>
          </template>
        </el-table-column>
        <el-table-column type="index" label="#" width="100" />
        <el-table-column prop="id" :label="$t('Device ID')" width="200" />
        <el-table-column prop="code" :label="$t('Status Code')" width="70" />
        <el-table-column prop="err" :label="$t('Error Code')" width="70" />
        <el-table-column prop="msg" :label="$t('Error Message')" width="200" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import { InternetExplorer as IEIcon } from '@vicons/fa'
import { Terminal as TerminalIcon } from '@vicons/ionicons5'

export default {
  name: 'Home',
  components: {
    IEIcon,
    TerminalIcon
  },
  data() {
    return {
      filterString: '',
      loading: true,
      devlists: [],
      filteredDevices: [],
      selection: [],
      currentPage: 1,
      pageSize: 10,
      cmdModal: false,
      inputParamVisible: false,
      inputParamValue: '',
      cmdStatus: {
        total: 0,
        modal: false,
        execing: 0,
        fail: 0,
        respModal: false,
        responses: []
      },
      cmdData: {
        username: '',
        cmd: '',
        params: [],
        currentParam: '',
        wait: 30
      },
      cmdRuleValidate: {
        username: [{required: true, message: this.$t('username is required')}],
        cmd: [{required: true, message: this.$t('command is required')}],
        wait: [{validator: (rule, value, callback) => {
          if (!value) {
            callback()
            return
          }

          if (!Number.isInteger(value) || value < 0 || value > 30) {
            callback(new Error(this.$t('must be an integer between 0 and 30')))
          }

          callback()
        }}]
      }
    }
  },
  computed: {
    cmdStatusPercent() {
      if (this.cmdStatus.total === 0)
        return 0
      return (this.cmdStatus.total - this.cmdStatus.execing) / this.cmdStatus.total * 100
    },
    pagedevlists() {
      return this.filteredDevices.slice((this.currentPage - 1) * this.pageSize, this.currentPage * this.pageSize)
    }
  },
  methods: {
    formatTime(t) {
      let ts = t || 0
      let tm = 0
      let th = 0
      let td = 0

      if (ts > 59) {
        tm = Math.floor(ts / 60)
        ts = ts % 60
      }

      if (tm > 59) {
        th = Math.floor(tm / 60)
        tm = tm % 60
      }

      if (th > 23) {
        td = Math.floor(th / 24)
        th = th % 24
      }

      let s = ''

      if (td > 0)
        s = `${td}d `

      return s + `${th}h ${tm}m ${ts}s`
    },
    handlePageChange(page, size) {
      this.currentPage = page
      this.pageSize = size
    },
    handleLogout() {
      this.axios.get('/signout').then(() => {
        this.$router.push('/login')
      })
    },
    handleSearch() {
      this.filteredDevices = this.devlists.filter((d) => {
        const filterString = this.filterString.toLowerCase()
        return d.id.toLowerCase().indexOf(filterString) > -1 || d.description.toLowerCase().indexOf(filterString) > -1
      })
    },
    getDevices() {
      this.axios.get('/devs').then(res => {
        this.loading = false
        this.devlists = res.data
        this.selection = []
        this.handleSearch()
      }).catch(() => {
        this.$router.push('/login')
      })
    },
    handleRefresh() {
      this.loading = true
      setTimeout(() => {
        this.getDevices()
      }, 500)
    },
    handleSelection(selection) {
      this.selection = selection
    },
    connectDevice(devid) {
      window.open('/rtty/' + devid)
    },
    isValidURL(url) {
      const ipv4Pattern = '(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)'
      const urlPattern = new RegExp(`^https?:\\/\\/${ipv4Pattern}(?::\\d{1,5})?(?:\\/[^\\s]*)?$`)
      return urlPattern.test(url)
    },
    parseURL(url) {
      const result = {
        proto: 'http',
        ip: '',
        port: '80',
        path: '/'
      }

      if (url.startsWith('https://')) {
        result.proto = 'https'
        url = url.substring(8)
        result.port = '443'
      } else if (url.startsWith('http://')) {
        url = url.substring(7)
      }

      const [address, ...pathParts] = url.split('/')

      result.path = pathParts.length > 0 ? '/' + pathParts.join('/') : '/'

      const [ip, port] = address.split(':')
      result.ip = ip
      if (port) {
        result.port = port
      }

      return result
    },
    connectDeviceWeb(dev) {
      this.$prompt(this.$t('Please enter the address you want to access'), '', {
        confirmButtonText: this.$t('OK'),
        cancelButtonText: this.$t('Cancel'),
        inputValue: 'http://127.0.0.1',
        inputValidator: this.isValidURL,
        inputErrorMessage: this.$t('Invalid Address')
      }).then(({ value }) => {
        const url = this.parseURL(value)

        if (dev.proto < 4 && url.proto === 'https') {
          this.$message.error(this.$t('Your device\'s rtty does not support https proxy, please upgrade it.'))
          return
        }

        const addr = encodeURIComponent(`${url.ip}:${url.port}${url.path}`)
        window.open(`/web/${dev.id}/${url.proto}/${addr}`)
      })
    },
    showCmdForm() {
      if (this.selection.length < 1) {
        this.$message.error(this.$t('Please select the devices you want to operate'))
        return
      }
      this.cmdModal = true
    },
    delCmdParam(tag) {
      this.cmdData.params.splice(this.cmdData.params.indexOf(tag), 1)
    },
    showInputParam() {
      this.inputParamVisible = true
      this.$nextTick(() => {
        (this.$refs.inputParam).focus()
      })
    },
    handleInputParamConfirm() {
      const value = this.inputParamValue
      if (value) {
        this.cmdData.params.push(value)
      }
      this.inputParamVisible = false
      this.inputParamValue = ''
    },
    doCmd() {
      (this.$refs['cmdForm']).validate(valid => {
        if (valid) {
          const selection = this.selection.filter(item => item.proto > 4)

          this.cmdModal = false
          this.cmdStatus.modal = true
          this.cmdStatus.total = selection.length
          this.cmdStatus.execing = selection.length
          this.cmdStatus.fail = 0
          this.cmdStatus.responses = []

          selection.forEach(item => {
            const data = {
              username: this.cmdData.username,
              cmd: this.cmdData.cmd,
              params: this.cmdData.params
            }

            this.axios.post(`/cmd/${item.id}?wait=${this.cmdData.wait}`, data).then((response) => {
              if (this.cmdData.wait === 0) {
                this.cmdStatus.responses.push({
                  err: 0,
                  msg: '',
                  id: item.id,
                  code: 0,
                  stdout: '',
                  stderr: ''
                })
              } else {
                const resp = response.data

                if (resp.err && resp.err !== 0) {
                  this.cmdStatus.fail++
                  resp.stdout = ''
                  resp.stderr = ''
                } else {
                  resp.stdout = window.atob(resp.stdout || '')
                  resp.stderr = window.atob(resp.stderr || '')
                }

                resp.id = item.id
                this.cmdStatus.responses.push(resp)
              }
              this.cmdStatus.execing--
            })
          })
        }
      })
    },
    resetCmdData() {
      (this.$refs['cmdForm']).resetFields()
    },
    ignoreCmdResp() {
      this.cmdStatus.execing = 0
      this.cmdStatus.respModal = true
      this.cmdStatus.modal = false
    },
    showCmdResp() {
      this.cmdStatus.modal = false
      if (this.cmdStatus.responses.length > 0)
        this.cmdStatus.respModal = true
    }
  },
  mounted() {
    this.getDevices()
  }
}
</script>

<style scoped>
  :deep(.el-pagination) .el-pagination__total {
    color: white;
  }

  :deep(.el-pagination) {
    justify-content: center;
  }
</style>
