<template>
  <div style="padding:5px;">
    <el-button style="margin-right: 4px;" type="primary" round icon="el-icon-refresh" @click="handleRefresh" :disabled="loading">{{$t('Refresh List')}}</el-button>
    <el-input style="margin-right: 4px;width:200px" v-model="filterString" suffix-icon="el-icon-search"
              @input="handleSearch" :placeholder="$t('Please enter the filter key...')"/>
    <el-button style="margin-right: 4px;" @click="showCmdForm" type="primary" :disabled="cmdStatus.execing > 0">{{$t('Execute command')}}</el-button>
    <el-button v-if="isadmin" style="margin-right: 4px;" @click="showBindForm" type="primary">{{$t('Bind user')}}</el-button>
    <el-tooltip :content="$t('Delete offline devices')">
      <el-button @click="deleteDevices" type="primary">{{$t('Delete')}}</el-button>
    </el-tooltip>
    <div style="float: right; margin-right: 10px">
      <span style="margin-right: 20px; color: #3399ff; font-size: 24px">{{ $t('device-count', {count: devlists.filter(dev => dev.online).length}) }}</span>
      <el-dropdown @command="handleUserCommand">
        <span class="el-dropdown-link">
          <span style="color: #3399ff; font-size: 24px">{{ username }}</span>
          <i class="el-icon-arrow-down el-icon--right" style="color: #3399ff; font-size: 24px"/>
        </span>
        <el-dropdown-menu slot="dropdown">
          <el-dropdown-item command="logout">{{ $t('Sign out') }}</el-dropdown-item>
        </el-dropdown-menu>
        </el-dropdown>
      </div>
    <el-table v-loading="loading" :data="filtered"
              style="margin-top: 10px; width: 100%" :empty-text="$t('No devices connected')"
              @selection-change='handleSelection'>
      <el-table-column type="index" label="#" width="100"/>
      <el-table-column type="selection"/>
      <el-table-column prop="id" :label="$t('Device ID')" sortable width="200"/>
      <el-table-column prop="connected" :label="$t('Connected time')" sortable width="200">
        <template v-slot="{ row }">
          <span v-if="row.online">{{ row.connected | formatTime }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="uptime" :label="$t('Uptime')" sortable width="200">
        <template v-slot="{ row }">
          <span v-if="row.online">{{ row.uptime | formatTime }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="description" :label="$t('Description')" show-overflow-tooltip/>
      <el-table-column label="#" width="200">
        <template v-slot="{ row }">
          <el-button v-if="isadmin && row.bound" type="danger" size="small" style="vertical-align: bottom;" @click="unBindUser(row.id)">{{ $t('Unbind') }}</el-button>
          <el-tooltip v-if="row.online" placement="top" :content="$t('Access your device\'s Shell')">
            <el-button @click="connectDevice(row.id)" style="padding: 0"><i class="iconfont icon-shell" style="font-size: 40px; color: black"/></el-button>
          </el-tooltip>
          <el-tooltip v-if="row.online" placement="top" :content="$t('Access your devices\'s Web')">
            <el-button @click="connectDeviceWeb(row.id)" style="padding: 0"><i class="iconfont icon-web" style="font-size: 40px; color: #409EFF"/></el-button>
          </el-tooltip>
          <span style="margin-left: 10px; color: red" v-if="!row.online">{{ $t('Device offline') }}</span>
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
          <el-input v-model.trim="cmdData.cmd"/>
        </el-form-item>
        <el-form-item :label="$t('Parameter')" prop="params">
          <el-tag :key="tag" v-for="tag in cmdData.params" closable @close="delCmdParam(tag)">{{tag}}</el-tag>
          <el-input style="width: 90px; margin-left: 10px;" v-if="inputParamVisible" v-model="inputParamValue"
                    ref="inputParam" size="small" @keyup.enter.native="handleInputParamConfirm"
                    @blur="handleInputParamConfirm"/>
          <el-button v-else style="width: 40px; margin-left: 10px;" size="small" icon="el-icon-plus" type="primary"
                     @click="showInputParam"/>
        </el-form-item>
        <el-form-item :label="$t('Wait Time')" prop="wait">
          <el-input v-model.number="cmdData.wait" placeholder="30">
            <template slot="append">s</template>
          </el-input>
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
    <el-dialog :visible.sync="bindUserData.modal" :title="$t('Bind user')" width="300px">
      <el-select v-model="bindUserData.currentUser">
        <el-option v-for="u in bindUserData.users" :key="u" :value="u"/>
      </el-select>
      <span slot="footer" class="dialog-footer">
        <el-button type="primary" :disabled="!bindUserData.currentUser" @click="bindUser">{{ $t('OK') }}</el-button>
      </span>
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
    bound: boolean;
    online: boolean;
  }

  interface ResponseInfo {
    id: string;
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
    username = '';
    isadmin = false;
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
      respModal: false,
      responses: [] as ResponseInfo[]
    };
    cmdData = {
      username: '',
      password: '',
      cmd: '',
      params: [] as string[],
      currentParam: '',
      wait: 30
    };
    bindUserData = {
      modal: false,
      users:[],
      currentUser: ''
    };
    cmdRuleValidate = {
      username: [{required: true, message: this.tr('username is required')}],
      cmd: [{required: true, message: this.tr('command is required')}],
      wait: [{validator: (rule, value, callback) => {
        if (!value) {
          callback()
          return;
        }

        if (!Number.isInteger(value) || value < 0 || value > 30) {
          callback(new Error(this.tr('must be an integer between 0 and 30')));
        }

        callback()
      }}]
    };

    tr(key: string): string {
      if (this.$t)
        return this.$t(key).toString();
      return '';
    }

    handleUserCommand(command: string) {
      if (command === 'logout') {
        this.axios.get('/signout').then(() => {
          this.$router.push('/login');
        });
      }
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

    showBindForm() {
      if (this.selection.length < 1) {
        this.$message.error(this.$t('Please select the devices you want to bind').toString());
        return;
      }

      this.axios.get('/users').then(res => {
        this.bindUserData.users = res.data.users;
        this.bindUserData.modal = true;
      });
    }

    bindUser() {
      this.bindUserData.modal = false;

      this.axios.post('/bind', {
        devices: this.selection.map(s => s.id),
        username: this.bindUserData.currentUser
      }).then(() => {
        this.getDevices();
        this.$message.success(this.$t('Bind success').toString());
      });
    }

    unBindUser(id: string) {
      this.axios.post('/unbind', {
        devices: [id]
      }).then(() => {
        this.getDevices();
        this.$message.success(this.$t('Unbind success').toString());
      });
    }

    deleteDevices() {
      if (this.selection.length < 1) {
        this.$message.error(this.$t('Please select the devices you want to operate').toString());
        return;
      }

      this.axios.post('/delete', {
        devices: this.selection.filter(s => !s.online).map(s => s.id)
      }).then(() => {
        this.getDevices();
        this.$message.success(this.$t('Delete success').toString());
      });
    }

    connectDevice(devid: string) {
      window.open('/rtty/' + devid);
    }

    connectDeviceWeb(devid: string) {
      this.$prompt(this.$t('Please enter the address you want to access').toString() + ':', this.$t('Access your devices\'s Web').toString(), {
        inputValue: '127.0.0.1:80',
        inputValidator: (value: string): boolean => {
          const ipreg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/
          const portreg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5]):\d{1,5}$/

          if (ipreg.test(value))
            return true;

          if (portreg.test(value)) {
            const port = Number(value.substr(value.lastIndexOf(':') + 1));
            return port > 0 && port <= 65535;
          }

          return false;
        }
      }).then((r => {
        const addr = encodeURIComponent((r as any).value)
        setTimeout(() => {
          window.open(`/web/${devid}/${addr}/`);
        }, 100)
      }));
    }

    showCmdForm() {
      if (this.selection.length < 1) {
        this.$message.error(this.$t('Please select the devices you want to operate').toString());
        return;
      }
      this.cmdModal = true;
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
          const selection = this.selection.filter(dev => dev.online);

          this.cmdModal = false;
          this.cmdStatus.modal = true;
          this.cmdStatus.total = selection.length;
          this.cmdStatus.execing = selection.length;
          this.cmdStatus.fail = 0;
          this.cmdStatus.responses = [];

          selection.forEach(item => {
            const data = {
              username: this.cmdData.username,
              password: this.cmdData.password,
              cmd: this.cmdData.cmd,
              params: this.cmdData.params
            };

            this.axios.post(`/cmd/${item.id}?wait=${this.cmdData.wait}`, data).then((response) => {
              if (this.cmdData.wait === 0) {
                this.cmdStatus.responses.push({
                  err: 0,
                  msg: '',
                  id: item.id,
                  code: 0,
                  stdout: '',
                  stderr: ''
                });
              } else {
                const resp = response.data as ResponseInfo;

                if (resp.err && resp.err !== 0) {
                    this.cmdStatus.fail++;
                    resp.stdout = '';
                    resp.stderr = '';
                } else {
                  resp.stdout = window.atob(resp.stdout || '');
                  resp.stderr = window.atob(resp.stderr || '');
                }

                resp.id = item.id;
                this.cmdStatus.responses.push(resp);
              }
              this.cmdStatus.execing--;
            });
          });
        }
      });
    }

    resetCmdData() {
      (this.$refs['cmdForm'] as ElForm).resetFields();
    }

    ignoreCmdResp() {
      this.cmdStatus.execing = 0;
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
      this.username = sessionStorage.getItem('rttys-username') || '';

      this.axios.get('/isadmin').then(res => {
        this.isadmin = res.data.admin;
      });

      this.getDevices();
    }
  }
</script>

<style scoped>
  .el-tag + .el-tag {
    margin-left: 10px;
  }
</style>
