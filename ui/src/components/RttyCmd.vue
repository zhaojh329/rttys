<template>
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
      <el-button type="danger" :disabled="cmdStatus.execing === 0" @click="ignoreCmdResp">{{ $t('Ignore') }}</el-button>
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
</template>

<script>
export default {
  name: 'RttyCmd',
  props: {
    selection: Array
  },
  data() {
    return {
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
    }
  },
  methods: {
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

            this.axios.post(`/cmd/${item.id}?group=${item.group}&wait=${this.cmdData.wait}`, data).then((response) => {
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
  }
}
</script>
