<template>
  <div style="padding:5px;">
    <el-button style="margin-right: 4px;" type="primary" round icon="el-icon-refresh" @click="handleRefresh" :disabled="loading">{{$t('Refresh List')}}</el-button>
    <el-input style="margin-right: 4px;width:200px" v-model="filterString" suffix-icon="el-icon-search"
              @input="handleSearch" :placeholder="$t('Please enter the filter key...')"/>
    <el-button style="margin-right: 4px;" @click="showCmdForm" type="primary" :disabled="cmdStatus.execing > 0">
      {{$t('Execute command')}}
    </el-button>
    <div style="float: right; color: #3399ff; font-size: 16px">{{ $t('device-count', {count: devlists.length}) }}</div>
    <el-table v-loading="loading" :data="filtered"
              style="margin-top: 10px; width: 100%" :empty-text="$t('No devices connected')"
              @selection-change='handleSelection'>
      <el-table-column type="index" label="#" width="100"/>
      <el-table-column type="selection"/>
      <el-table-column prop="id" :label="$t('Device ID')" sortable width="300"/>
      <el-table-column prop="uptime" :label="$t('Uptime')" sortable width="150">
        <template v-slot="{ row }">{{ row.uptime | formatTime }}</template>
      </el-table-column>
      <el-table-column prop="description" :label="$t('Description')" show-overflow-tooltip/>
      <el-table-column label="#" width="150">
        <template v-slot="{ row }">
          <el-button type="primary" @click="connectDevice(row.id)">{{ $t('Connect') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog :visible.sync="cmdModal" :title="$t('Execute command')" width="600px">
      <el-form ref="cmdForm" :model="cmdData" :rules="cmdRuleValidate" label-width="100px" label-position="left">
        <el-form-item :label="$t('Username')" prop="username">
          <el-input v-model="cmdData.username"/>
        </el-form-item>
        <el-form-item :label="$t('Password')" prop="password">
          <el-input v-model="cmdData.password" show-password/>
        </el-form-item>
        <el-form-item :label="$t('Command')" prop="cmd">
          <el-input v-model="cmdData.cmd"/>
        </el-form-item>
        <el-form-item :label="$t('Parameter')" prop="params">
          <el-tag :key="tag" v-for="tag in cmdData.params" closable @close="delCmdParam(tag)">{{tag}}</el-tag>
          <el-input style="width: 90px; margin-left: 10px;" v-if="inputParamVisible" v-model="inputParamValue"
                    ref="inputParam" size="small" @keyup.enter.native="handleInputParamConfirm"
                    @blur="handleInputParamConfirm"/>
          <el-button v-else style="width: 40px; margin-left: 10px;" size="small" icon="el-icon-plus" type="primary"
                     @click="showInputParam"/>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" style="width: 70%" @click="doCmd">{{ $t('OK') }}</el-button>
          <el-button type="warning" @click="resetCmdData">{{ $t('Reset') }}</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
    <el-dialog :visible.sync="cmdStatus.modal" :title="$t('Status of executive command')" :show-close="false"
               :close-on-press-escape="false" :close-on-click-modal="false">
      <el-progress :text-inside="true" :stroke-width="26" :percentage="cmdStatusPercent"/>
      <p>{{ $t('cmd-status-total', {count: cmdStatus.total}) }}</p>
      <p>{{ $t('cmd-status-fail', {count: cmdStatus.fail}) }}</p>
      <div slot="footer">
        <el-button type="primary" size="large" :disabled="cmdStatus.execing > 0" @click="showCmdResp">{{$t('OK')}}</el-button>
        <el-button type="danger" size="large" :disabled="cmdStatus.execing === 0" @click="ignoreCmdResp">{{$t('Ignore')}}</el-button>
      </div>
    </el-dialog>
    <el-dialog :visible.sync="cmdStatus.respModal" :title="$t('Response of executive command')" width="1000">
      <el-table :data="cmdStatus.responses" height="300" :empty-text="$t('No Response')">
        <el-table-column type="index" label="#"/>
        <el-table-column prop="id" :label="$t('Device ID')"/>
        <el-table-column prop="cmd" :label="$t('Command')"/>
        <el-table-column prop="err" :label="$t('Error Code')"/>
        <el-table-column prop="msg" :label="$t('Error Message')" show-overflow-tooltip/>
        <el-table-column prop="code" :label="$t('Status Code')"/>
        <el-table-column width="150">
          <template slot="header">
            <span>{{ $t('Stdout') }}</span><br/>
            <span>{{ $t('(Show all by mouse hover)') }}</span>
          </template>
          <template v-slot="{ row }">
            <el-tooltip placement="top">
              <pre slot="content">{{ row.stdout }}</pre>
              <span>{{ row.stdout.substr(0, 5) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column width="150">
          <template slot="header">
            <span>{{ $t('Stderr') }}</span><br/>
            <span>{{ $t('(Show all by mouse hover)') }}</span>
          </template>
          <template v-slot="{ row }">
            <el-tooltip placement="top">
              <pre slot="content">{{ row.stderr }}</pre>
              <span>{{ row.stderr.substr(0, 5) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
      <div slot="footer"></div>
    </el-dialog>
  </div>
</template>

<script lang="ts">
  import {Component, Vue} from 'vue-property-decorator'
  import {Form as ElForm, Input as ElInput} from 'element-ui/types/element-ui';

  interface DeviceInfo {
    id: string;
    uptime: number;
    description: string;
  }

  interface CmdStatusInfo {
    querying: boolean;
    devid: string;
    cmd: string;
  }

  interface ResponseInfo {
    token: string;
    devid: string;
    cmd: string;
    code: number;
    err: number;
    msg: string;
    stdout: string;
    stderr: string;
  }

  @Component({
    filters: {
      formatTime(t: number) {
        let ts = t || 0;
        let tm = 0;
        let th = 0;
        let td = 0;

        if (ts > 59) {
          tm = Math.floor(ts / 60);
          ts = ts % 60;
        }

        if (tm > 59) {
          th = Math.floor(tm / 60);
          tm = tm % 60;
        }

        if (th > 23) {
          td = Math.floor(th / 24);
          th = th % 24;
        }

        let s = '';

        if (td > 0)
          s = `${td}d `;

        return s + `${th}h ${tm}m ${ts}s`;
      }
    }
  })
  export default class Home extends Vue {
    filterString = '';
    loading = true;
    devlists = [];
    filtered = [];
    selection = [] as DeviceInfo[];
    cmdModal = false;
    inputParamVisible = false;
    inputParamValue = '';
    cmdStatus = {
      total: 0,
      modal: false,
      execing: 0,
      fail: 0,
      running: {} as { [key: string]: CmdStatusInfo },
      respModal: false,
      responses: [] as ResponseInfo[]
    };
    cmdData = {
      username: '',
      password: '',
      cmd: '',
      params: [] as string[],
      currentParam: ''
    };
    cmdRuleValidate = {
      username: [{required: true, trigger: 'blur', message: this.tr('username is required')}],
      cmd: [{required: true, trigger: 'blur', message: this.tr('command is required')}]
    };

    tr(key: string): string {
      if (this.$t)
        return this.$t(key).toString();
      return '';
    }

    handleSearch() {
      this.filtered = this.devlists.filter((d: DeviceInfo) => {
        const filterString = this.filterString.toLowerCase();
        return d.id.toLowerCase().indexOf(filterString) > -1 || d.description.toLowerCase().indexOf(filterString) > -1;
      });
    }

    getDevices() {
      this.axios.get('/devs').then(res => {
        this.loading = false;
        this.devlists = res.data;
        this.handleSearch();
      }).catch(() => {
        this.$router.push('/login');
      });
    }

    handleRefresh() {
      this.loading = true;
      setTimeout(() => {
        this.getDevices();
      }, 500);
    }

    handleSelection(selection: DeviceInfo[]) {
      this.selection = selection;
    }

    connectDevice(devid: string) {
      this.$router.push({path: `/rtty/${devid}`});
    }

    showCmdForm() {
      if (this.selection.length < 1) {
        this.$message.error(this.$t('Please select the devices you want to operate').toString());
        return;
      }
      this.cmdModal = true;
    }

    queryCmdResp() {
      let count = 0;

      for (const token in this.cmdStatus.running) {
        const item = this.cmdStatus.running[token];

        if (item.querying)
          continue;

        item.querying = true;

        this.axios.get(`/cmd/${item.devid}/${token}`).then(response => {
          const resp = response.data as ResponseInfo;

          if (resp.err === 1005) {
            item.querying = false;
            return;
          }

          if (resp.err && resp.err !== 0)
            this.cmdStatus.fail++;

          this.cmdStatus.execing--;

          resp.devid = item.devid;
          resp.cmd = item.cmd;
          resp.stdout = window.atob(resp.stdout || '');
          resp.stderr = window.atob(resp.stderr || '');

          this.cmdStatus.responses.push(resp);

          delete this.cmdStatus.running[token];
        });

        count++;

        if (count > 10)
          break;
      }

      if (this.cmdStatus.execing > 0)
        setTimeout(this.queryCmdResp, 500);
    }

    delCmdParam(tag: string) {
      this.cmdData.params.splice(this.cmdData.params.indexOf(tag), 1);
    }

    showInputParam() {
      this.inputParamVisible = true;
      this.$nextTick(() => {
        (this.$refs.inputParam as ElInput).focus();
      });
    }

    handleInputParamConfirm() {
      const value = this.inputParamValue;
      if (value) {
        this.cmdData.params.push(value);
      }
      this.inputParamVisible = false;
      this.inputParamValue = '';
    }

    doCmd() {
      (this.$refs['cmdForm'] as ElForm).validate(valid => {
        if (valid) {
          this.cmdModal = false;
          this.cmdStatus.modal = true;
          this.cmdStatus.total = this.selection.length;
          this.cmdStatus.execing = this.selection.length;
          this.cmdStatus.fail = 0;
          this.cmdStatus.running = {};
          this.cmdStatus.responses = [];

          this.selection.forEach(item => {
            const data = {
              username: this.cmdData.username,
              password: this.cmdData.password,
              sid: sessionStorage.getItem('rtty-sid'),
              cmd: this.cmdData.cmd.trim(),
              params: this.cmdData.params
            };

            this.axios.post(`/cmd/${item.id}`, data).then((response) => {
              const resp = response.data as ResponseInfo;

              if (resp.token) {
                this.cmdStatus.running[resp.token] = {
                  devid: item.id,
                  cmd: data.cmd,
                  querying: false
                };
                return;
              }

              this.cmdStatus.execing--;
              this.cmdStatus.fail++;

              resp.devid = item.id;
              resp.cmd = data.cmd;

              this.cmdStatus.responses.push(resp);
            });
          });

          setTimeout(this.queryCmdResp, 100);
        }
      });
    }

    resetCmdData() {
      (this.$refs['cmdForm'] as ElForm).resetFields();
    }

    ignoreCmdResp() {
      this.cmdStatus.execing = 0;
      this.cmdStatus.running = {};

      this.cmdStatus.respModal = true;
      this.cmdStatus.modal = false;
    }

    showCmdResp() {
      this.cmdStatus.modal = false;
      if (this.cmdStatus.responses.length > 0)
        this.cmdStatus.respModal = true;
    }

    get cmdStatusPercent(): number {
      if (this.cmdStatus.total === 0)
        return 0;
      return (this.cmdStatus.total - this.cmdStatus.execing) / this.cmdStatus.total * 100;
    }

    mounted() {
      this.getDevices();
    }
  }
</script>

<style scoped>
  .el-tag + .el-tag {
    margin-left: 10px;
  }
</style>
