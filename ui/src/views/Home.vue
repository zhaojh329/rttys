<template>
  <div style="padding:5px;">
    <div style="display: flex; justify-content: space-between;">
      <el-space>
        <el-button type="primary" round icon="Refresh" @click="handleRefresh" :disabled="loading">{{ $t('Refresh List') }}</el-button>
        <el-input style="width:200px" v-model="filterString" search @input="handleSearch" :placeholder="$t('Please enter the filter key...')"/>
        <el-button @click="showCmdForm" type="primary">{{ $t('Execute command') }}</el-button>
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
    <RttyCmd ref="rttyCmd" :selection="selection"/>
  </div>
</template>

<script>
import { InternetExplorer as IEIcon } from '@vicons/fa'
import { Terminal as TerminalIcon } from '@vicons/ionicons5'
import RttyCmd from '../components/RttyCmd.vue'

export default {
  name: 'Home',
  components: {
    IEIcon,
    TerminalIcon,
    RttyCmd
  },
  data() {
    return {
      filterString: '',
      loading: true,
      devlists: [],
      filteredDevices: [],
      selection: [],
      currentPage: 1,
      pageSize: 10
    }
  },
  computed: {
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
      this.$refs.rttyCmd.showCmdForm()
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
