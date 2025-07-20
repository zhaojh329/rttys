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

<script setup>
import { ref, reactive, computed, nextTick, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import axios from 'axios'

const { t } = useI18n()

const props = defineProps({
  selection: Array
})

const cmdForm = useTemplateRef('cmdForm')
const inputParam = useTemplateRef('inputParam')

const cmdModal = ref(false)
const inputParamVisible = ref(false)
const inputParamValue = ref('')

const cmdStatus = reactive({
  total: 0,
  modal: false,
  execing: 0,
  fail: 0,
  respModal: false,
  responses: []
})

const cmdData = reactive({
  username: '',
  cmd: '',
  params: [],
  currentParam: '',
  wait: 30
})

const cmdRuleValidate = {
  username: [{required: true, message: t('username is required')}],
  cmd: [{required: true, message: t('command is required')}],
  wait: [{validator: (rule, value, callback) => {
    if (!value) {
      callback()
      return
    }

    if (!Number.isInteger(value) || value < 0 || value > 30) {
      callback(new Error(t('must be an integer between 0 and 30')))
    }

    callback()
  }}]
}

const cmdStatusPercent = computed(() => {
  if (cmdStatus.total === 0)
    return 0
  return (cmdStatus.total - cmdStatus.execing) / cmdStatus.total * 100
})

const showCmdForm = () => {
  if (props.selection.length < 1) {
    ElMessage.error(t('Please select the devices you want to operate'))
    return
  }
  cmdModal.value = true
}

const delCmdParam = (tag) => cmdData.params.splice(cmdData.params.indexOf(tag), 1)

const showInputParam = () => {
  inputParamVisible.value = true
  nextTick(() => inputParam.value.focus())
}

const handleInputParamConfirm = () => {
  const value = inputParamValue.value
  if (value) {
    cmdData.params.push(value)
  }
  inputParamVisible.value = false
  inputParamValue.value = ''
}

const doCmd = () => {
  cmdForm.value.validate(valid => {
    if (valid) {
      const selection = props.selection.filter(item => item.proto > 4)

      cmdModal.value = false
      cmdStatus.modal = true
      cmdStatus.total = selection.length
      cmdStatus.execing = selection.length
      cmdStatus.fail = 0
      cmdStatus.responses = []

      selection.forEach(item => {
        const data = {
          username: cmdData.username,
          cmd: cmdData.cmd,
          params: cmdData.params
        }

        axios.post(`/cmd/${item.id}?group=${item.group}&wait=${cmdData.wait}`, data).then((response) => {
          if (cmdData.wait === 0) {
            cmdStatus.responses.push({
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
              cmdStatus.fail++
              resp.stdout = ''
              resp.stderr = ''
            } else {
              resp.stdout = window.atob(resp.stdout || '')
              resp.stderr = window.atob(resp.stderr || '')
            }

            resp.id = item.id
            cmdStatus.responses.push(resp)
          }
          cmdStatus.execing--
        })
      })
    }
  })
}

const resetCmdData = () => cmdForm.value.resetFields()

const ignoreCmdResp = () => {
  cmdStatus.execing = 0
  cmdStatus.respModal = true
  cmdStatus.modal = false
}

const showCmdResp = () => {
  cmdStatus.modal = false
  if (cmdStatus.responses.length > 0)
    cmdStatus.respModal = true
}

defineExpose({
  showCmdForm,
  resetCmdData
})
</script>
