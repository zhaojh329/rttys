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
        <el-table height="calc(100vh - 225px)" :loading="loading" :data="pagedevlists" :empty-text="$t('No devices connected')" @selection-change='handleSelection'>
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
      <div class="footer">
        <el-text type="info">Powered by </el-text>
        <el-link type="primary" href="https://github.com/zhaojh329/rtty" target="_blank">rtty</el-link>
      </div>
    </el-main>
    <RttyCmd ref="rttyCmd" :selection="selection"/>
    <RttyWeb v-model="web.modal" :dev="web.dev"/>
  </el-container>
</template>

<script setup>
import { ref, reactive, computed, onMounted, useTemplateRef } from 'vue'
import { useRouter } from 'vue-router'
import { InternetExplorer as IEIcon } from '@vicons/fa'
import { Terminal as TerminalIcon } from '@vicons/ionicons5'
import RttyCmd from '../components/RttyCmd.vue'
import RttyWeb from '../components/RttyWeb.vue'
import axios from 'axios'

const router = useRouter()

const rttyCmd = useTemplateRef('rttyCmd')

const group = ref('')
const groups = ref([])
const filterString = ref('')
const loading = ref(true)
const devlists = ref([])
const filteredDevices = ref([])
const selection = ref([])
const currentPage = ref(1)
const pageSize = ref(10)
const web = reactive({
  modal: false,
  dev: null
})

const pagedevlists = computed(() => {
  return filteredDevices.value.slice((currentPage.value - 1) * pageSize.value, currentPage.value * pageSize.value)
})

const formatTime = (t) => {
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
}

const handlePageChange = (page, size) => {
  currentPage.value = page
  pageSize.value = size
}

const handleLogout = () => {
  axios.get('/signout').then(() => {
    router.push('/login')
  })
}

const handleSearch = () => {
  filteredDevices.value = devlists.value.filter((d) => {
    const filterStr = filterString.value.toLowerCase()
    return d.id.toLowerCase().indexOf(filterStr) > -1 || d.description.toLowerCase().indexOf(filterStr) > -1
  })
}

const getGroups = () => {
  axios.get('/groups').then(res => {
    groups.value = res.data
    if (groups.value.indexOf(group.value) === -1)
      group.value = groups.value[0]
    getDevices()
  })
}

const getDevices = () => {
  axios.get(`/devs?group=${group.value}`).then(res => {
    loading.value = false
    devlists.value = res.data
    selection.value = []
    handleSearch()
  }).catch(() => {
    router.push('/login')
  })
}

const handleRefresh = () => {
  loading.value = true
  setTimeout(() => getGroups(), 500)
}

const handleSelection = (sel) => selection.value = sel

const connectDevice = (devid) => {
  let url = `/rtty/${devid}`
  if (group.value)
    url += `?group=${group.value}`
  window.open(url)
}

const connectDeviceWeb = (dev) => {
  web.dev = dev
  web.modal = true
}

const showCmdForm = () => rttyCmd.value.showCmdForm()

onMounted(() => getGroups())
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
}

.footer {
  float: right;
  padding-right: 20px;
  height: 10px;
}

.pagination {
  display: flex;
  justify-content: center;
}
</style>
