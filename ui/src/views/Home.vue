<template>
  <el-container>
    <el-header class="header">
      <el-space>
        <el-button type="primary" round icon="Refresh" @click="handleRefresh" :disabled="loading">{{ $t('Refresh List') }}</el-button>
        <el-select v-model="group" filterable :placeholder="$t('ungrouped')" @change="getDevices">
          <el-option v-for="item in groups" :key="item" :label="item == '' ? $t('ungrouped'): item " :value="item"/>
        </el-select>
        <el-input style="width:200px" v-model="filterString" search @input="handleSearch" :placeholder="$t('Please enter the filter key...')"/>
        <el-button @click="showCmdForm" type="primary">{{ $t('Execute command') }}</el-button>
      </el-space>
      <el-space>
        <span style="color: var(--el-color-primary); font-size: 24px">{{ $t('device-count', {count: devlists.length}) }}</span>
        <el-divider direction="vertical" />
        <el-button type="primary" @click="handleLogout">{{ $t('Sign out') }}</el-button>
      </el-space>
    </el-header>
    <el-main>
      <el-card>
        <el-table height="calc(100vh - 215px)" :loading="loading" :data="pagedevlists" :empty-text="$t('No devices connected')" @selection-change='handleSelection'>
          <el-table-column type="selection" width="40" />
          <el-table-column prop="id" :label="$t('Device ID')" width="300" />
          <el-table-column :label="$t('Connected time')" width="150">
            <template #default="{ row }">
              <span>{{ formatTime(row.connected) }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('Uptime')" width="150">
            <template #default="{ row }">
              <span>{{ formatTime(row.uptime) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="ipaddr" :label="$t('ipaddr')" width="150" />
          <el-table-column prop="description" :label="$t('Description')" show-overflow-tooltip width="150" />
          <el-table-column width="100">
            <template #default="{ row }">
              <el-space size="large">
                <el-icon size="25" color="black" style="cursor:pointer;" @click="connectDevice(row.id)"><TerminalIcon /></el-icon>
                <el-icon size="25" color="#409EFF" style="cursor:pointer;" @click="connectDeviceWeb(row)"><IEIcon /></el-icon>
              </el-space>
            </template>
          </el-table-column>
        </el-table>
        <template #footer>
          <el-pagination background layout="prev, pager, next, total,sizes" :total="filteredDevices.length" @change="handlePageChange" class="pagination"/>
        </template>
      </el-card>
    </el-main>
    <RttyCmd ref="rttyCmd" :selection="selection"/>
    <RttyWeb ref="rttyWeb"/>
  </el-container>
</template>

<script>
import { InternetExplorer as IEIcon } from '@vicons/fa'
import { Terminal as TerminalIcon } from '@vicons/ionicons5'
import RttyCmd from '../components/RttyCmd.vue'
import RttyWeb from '../components/RttyWeb.vue'

export default {
  name: 'Home',
  components: {
    IEIcon,
    TerminalIcon,
    RttyCmd,
    RttyWeb
  },
  data() {
    return {
      group: '',
      groups: [],
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
    getGroups() {
      this.axios.get('/groups').then(res => {
        this.groups = res.data
        if (this.groups.indexOf(this.group) === -1)
          this.group = this.groups[0]
        this.getDevices()
      })
    },
    getDevices() {
      this.axios.get(`/devs?group=${this.group}`).then(res => {
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
        this.getGroups()
      }, 500)
    },
    handleSelection(selection) {
      this.selection = selection
    },
    connectDevice(devid) {
      let url = `/rtty/${devid}`
      if (this.group)
        url += `?group=${this.group}`
      window.open(url)
    },
    connectDeviceWeb(dev) {
      this.$refs.rttyWeb.show(dev)
    },
    showCmdForm() {
      this.$refs.rttyCmd.showCmdForm()
    }
  },
  mounted() {
    this.getGroups()
  }
}
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
}

.pagination {
  display: flex;
  justify-content: center;
}
</style>
