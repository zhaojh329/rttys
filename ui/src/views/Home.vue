<template>
  <div style="padding:5px;">
    <Button style="margin-right: 4px;" type="primary" shape="circle" icon="ios-refresh" @click="handleRefresh" :disabled="loading">{{ $t('Refresh List') }}</Button>
    <Input style="margin-right: 4px;width:200px" v-model="filterString" search @input="handleSearch" :placeholder="$t('Please enter the filter key...')"/>
    <Button style="margin-right: 4px;" @click="showCmdForm" type="primary" :disabled="cmdStatus.execing > 0">{{ $t('Execute command') }}</Button>
    <Button v-if="isadmin" style="margin-right: 4px;" @click="showBindForm" type="primary">{{ $t('Bind user') }}</Button>
    <Tooltip :content="$t('Delete offline devices')">
      <Button @click="deleteDevices" type="primary">{{ $t('Delete') }}</Button>
    </Tooltip>
    <div style="float: right; margin-right: 10px">
      <span style="margin-right: 20px; color: #3399ff; font-size: 24px">{{ $t('device-count', {count: devlists.filter(dev => dev.online).length}) }}</span>
      <Dropdown @on-click="handleUserCommand">
        <a href="javascript:void(0)">
            <span style="color: #3399ff; font-size: 24px">{{ username }}</span>
            <Icon type="ios-arrow-down"/>
        </a>
        <DropdownMenu slot="list">
          <DropdownItem name="logout">{{ $t('Sign out') }}</DropdownItem>
        </DropdownMenu>
      </Dropdown>
    </div>
    <Table :loading="loading" :columns="columnsDevices" :data="filteredDevices" style="margin-top: 10px; width: 100%" :no-data-text="$t('No devices connected')" @on-selection-change='handleSelection'>
      <template v-slot:connected="{ row }">
        <span v-if="row.online">{{ row.connected | formatTime }}</span>
      </template>
      <template v-slot:uptime="{ row }">
        <span v-if="row.online">{{ row.uptime  | formatTime }}</span>
      </template>
      <template v-slot:action="{ row }">
        <Button v-if="isadmin && row.bound" type="warning" size="small" style="vertical-align: bottom;" @click="unBindUser(row.id)">{{ $t('Unbind') }}</Button>
        <Tooltip v-if="row.online" placement="top" :content="$t('Access your device\'s Shell')">
          <i class="iconfont icon-shell" style="font-size: 40px; color: black; cursor:pointer;" @click="connectDevice(row.id)"/>
        </Tooltip>
        <Tooltip v-if="row.online" placement="top" :content="$t('Access your devices\'s Web')">
          <i class="iconfont icon-web" style="font-size: 40px; color: #409EFF; cursor:pointer;" @click="connectDeviceWeb(row)"/>
        </Tooltip>
        <span v-if="!row.online" style="margin-left: 10px; color: red">{{ $t('Device offline') }}</span>
      </template>
    </Table>
    <Modal v-model="cmdModal" :title="$t('Execute command')" @on-ok="doCmd">
      <Form ref="cmdForm" :model="cmdData" :rules="cmdRuleValidate" :label-width="100" label-position="left">
        <FormItem :label="$t('Username')" prop="username">
          <Input v-model="cmdData.username"/>
        </FormItem>
        <FormItem :label="$t('Password')" prop="password">
          <Input type="password" v-model="cmdData.password" password/>
        </FormItem>
        <FormItem :label="$t('Command')" prop="cmd">
          <Input v-model.trim="cmdData.cmd"/>
        </FormItem>
        <FormItem :label="$t('Parameter')" prop="params">
          <Tag :key="i" v-for="(tag, i) in cmdData.params" closable @on-close="delCmdParam(tag)">{{ tag }}</Tag>
          <Input v-if="inputParamVisible" style="width: 90px; margin-left: 10px;" v-model="inputParamValue"
                ref="inputParam" size="small" @on-enter="handleInputParamConfirm" @on-blur="handleInputParamConfirm"/>
          <Button v-else style="width: 40px; margin-left: 10px;" size="small" icon="ios-add" type="primary" @click="showInputParam"/>
        </FormItem>
        <FormItem :label="$t('Wait Time')" prop="wait">
          <Input v-model.number="cmdData.wait" placeholder="30">
            <template slot="append">s</template>
          </Input>
        </FormItem>
      </Form>
    </Modal>
    <Modal v-model="cmdStatus.modal" :title="$t('Status of executive command')" :closable="false" :mask-closable="false">
      <Progress text-inside :stroke-width="20" :percent="cmdStatusPercent"/>
      <p>{{ $t('cmd-status-total', {count: cmdStatus.total}) }}</p>
      <p>{{ $t('cmd-status-fail', {count: cmdStatus.fail}) }}</p>
      <div slot="footer">
        <Button type="primary" size="large" :disabled="cmdStatus.execing > 0" @click="showCmdResp">{{ $t('OK') }}</Button>
        <Button type="error" size="large" :disabled="cmdStatus.execing === 0" @click="ignoreCmdResp">{{ $t('Ignore') }}</Button>
      </div>
    </Modal>
    <Modal v-model="cmdStatus.respModal" :title="$t('Response of executive command')" :width="800">
      <Table :columns="columnsCmdResp" :data="cmdStatus.responses" :no-data-text="$t('No Response')"/>
      <div slot="footer"/>
    </Modal>
    <Modal v-model="bindUserData.modal" :title="$t('Bind user')" :width="300">
      <Select v-model="bindUserData.currentUser">
        <Option v-for="u in bindUserData.users" :key="u" :value="u"/>
      </Select>
      <span slot="footer" class="dialog-footer">
        <Button type="primary" :disabled="!bindUserData.currentUser" @click="bindUser">{{ $t('OK') }}</Button>
      </span>
    </Modal>
  </div>
</template>

<script>
import ExpandCmdResp from '@/components/ExpandCmdResp'

export default {
  name: 'Home',
  data() {
    return {
      username: '',
      isadmin: false,
      filterString: '',
      loading: true,
      devlists: [],
      filteredDevices: [],
      selection: [],
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
        password: '',
        cmd: '',
        params: [],
        currentParam: '',
        wait: 30
      },
      bindUserData: {
        modal: false,
        users:[],
        currentUser: ''
      },
      cmdRuleValidate: {
        username: [{required: true, message: this.$t('username is required')}],
        cmd: [{required: true, message: this.$t('command is required')}],
        wait: [{validator: (rule, value, callback) => {
          if (!value) {
            callback()
            return;
          }

          if (!Number.isInteger(value) || value < 0 || value > 30) {
            callback(new Error(this.$t('must be an integer between 0 and 30')));
          }

          callback()
        }}]
      },
      columnsDevices: [
        {
          title: '#',
          type: 'index',
          width: 100
        },
        {
          type: 'selection',
          width: 60
        },
        {
          title: this.$t('Device ID'),
          key: 'id',
          width: 200
        },
        {
          title: this.$t('Connected time'),
          slot: 'connected',
          width: 200
        },
        {
          title: this.$t('Uptime'),
          slot: 'uptime',
          width: 200
        },
        {
          title: this.$t('Description'),
          key: 'description',
          tooltip: true
        },
        {
          slot: 'action',
          width: 200
        }
      ],
      columnsCmdResp: [
        {
          type: 'expand',
          width: 50,
          render: (h, params) => {
            return h(ExpandCmdResp, {
              props: {
                stdout: params.row.stdout,
                stderr: params.row.stderr
              }
            })
          }
        },
        {
          title: '#',
          type: 'index',
          width: 100
        },
        {
          title: this.$t('Device ID'),
          key: 'id'
        },
        {
          title: this.$t('Status Code'),
          key: 'code'
        },
        {
          title: this.$t('Error Code'),
          key: 'err'
        },
        {
          title: this.$t('Error Message'),
          key: 'msg'
        }
      ]
    }
  },
  filters: {
    formatTime(t) {
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
  },
  computed: {
    cmdStatusPercent() {
      if (this.cmdStatus.total === 0)
        return 0;
      return (this.cmdStatus.total - this.cmdStatus.execing) / this.cmdStatus.total * 100;
    }
  },
  methods: {
    parseIPv4(x) {
      if (!x.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/))
        return null;

      if (RegExp.$1 > 255 || RegExp.$2 > 255 || RegExp.$3 > 255 || RegExp.$4 > 255)
        return null;

      return [ +RegExp.$1, +RegExp.$2, +RegExp.$3, +RegExp.$4 ];
    },
    handleUserCommand(command) {
      if (command === 'logout') {
        this.axios.get('/signout').then(() => {
          this.$router.push('/login');
        });
      }
    },
    handleSearch() {
      this.filteredDevices = this.devlists.filter((d) => {
        const filterString = this.filterString.toLowerCase();
        return d.id.toLowerCase().indexOf(filterString) > -1 || d.description.toLowerCase().indexOf(filterString) > -1;
      });
    },
    getDevices() {
      this.axios.get('/devs').then(res => {
        this.loading = false;
        this.devlists = res.data;
        this.selection = [];
        this.handleSearch();
      }).catch(() => {
        this.$router.push('/login');
      });
    },
    handleRefresh() {
      this.loading = true;
      setTimeout(() => {
        this.getDevices();
      }, 500);
    },
    handleSelection(selection) {
      this.selection = selection;
    },
    showBindForm() {
      if (this.selection.length < 1) {
        this.$Message.error(this.$t('Please select the devices you want to bind'));
        return;
      }

      this.axios.get('/users').then(res => {
        this.bindUserData.users = res.data.users;
        this.bindUserData.modal = true;
      });
    },
    bindUser() {
      this.bindUserData.modal = false;

      this.axios.post('/bind', {
        devices: this.selection.map(s => s.id),
        username: this.bindUserData.currentUser
      }).then(() => {
        this.getDevices();
        this.$Message.success(this.$t('Bind success'));
      });
    },
    unBindUser(id) {
      this.axios.post('/unbind', {
        devices: [id]
      }).then(() => {
        this.getDevices();
        this.$Message.success(this.$t('Unbind success'));
      });
    },
    deleteDevices() {
      if (this.selection.length < 1) {
        this.$Message.error(this.$t('Please select the devices you want to operate'));
        return;
      }

      const offlines = this.selection.filter(s => !s.online);

      if (offlines.length < 1) {
        this.$Message.info(this.$t('There are no offline devices in selected devices'));
        return;
      }

      this.axios.post('/delete', {
        devices: offlines.map(s => s.id)
      }).then(() => {
        this.getDevices();
        this.$Message.success(this.$t('Delete success'));
      });
    },
    connectDevice(devid) {
      window.open('/rtty/' + devid);
    },
    connectDeviceWeb(dev) {
      let addr = '127.0.0.1';

      this.$Modal.confirm({
        render: h => {
          const input = h('Input', {
              props: {
                value: addr,
                autofocus: true,
                placeholder: this.$t('Please enter the address you want to access')
              },
              on: {
                input: val => {
                  addr = val;
                }
              }
          });
          return h('div', [
            input,
            h('p', '127.0.0.1, 127.0.0.1:8080, 127.0.0.1/test.html?a=1'),
            h('p', 'http://127.0.0.1, https://127.0.0.1')
          ]);
        },
        onOk: () => {
          let proto = 'http';

          if (addr.startsWith('http://'))
            addr = addr.substring(7);

          if (addr.startsWith('https://')) {
            addr = addr.substring(8);
            proto = 'https';
          }

          if (dev.proto < 4 && proto === 'https') {
            this.$Message.error(this.$t('Your device\'s rtty does not support https proxy, please upgrade it.'));
            return;
          }

          let [addrs, ...path] = addr.split('/');

          path = '/' + path.join('/');

          let [ip, ...port] = addrs.split(':');

          if (!this.parseIPv4(ip)) {
            this.$Message.error(this.$t('Invalid address'));
            return;
          }

          if (port.length !== 0 && port.length !== 1) {
            this.$Message.error(this.$t('Invalid port'));
            return;
          }

          if (port.length === 1) {
            port = Number(port[0]);
            if (port <= 0 || port > 65535) {
              this.$Message.error(this.$t('Invalid port'));
              return;
            }
          } else {
            port = 80;
            if (proto === 'https')
              port = 443;
          }

          addr = encodeURIComponent(`${ip}:${port}${path}`);
          window.open(`/web/${dev.id}/${proto}/${addr}`);
        }
      });
    },
    showCmdForm() {
      if (this.selection.length < 1) {
        this.$Message.error(this.$t('Please select the devices you want to operate'));
        return;
      }
      this.cmdModal = true;
    },
    delCmdParam(tag) {
      this.cmdData.params.splice(this.cmdData.params.indexOf(tag), 1);
    },
    showInputParam() {
      this.inputParamVisible = true;
      this.$nextTick(() => {
        (this.$refs.inputParam).focus();
      });
    },
    handleInputParamConfirm() {
      const value = this.inputParamValue;
      if (value) {
        this.cmdData.params.push(value);
      }
      this.inputParamVisible = false;
      this.inputParamValue = '';
    },
    doCmd() {
      (this.$refs['cmdForm']).validate(valid => {
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
                const resp = response.data;

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
    },
    resetCmdData() {
      (this.$refs['cmdForm']).resetFields();
    },
    ignoreCmdResp() {
      this.cmdStatus.execing = 0;
      this.cmdStatus.respModal = true;
      this.cmdStatus.modal = false;
    },
    showCmdResp() {
      this.cmdStatus.modal = false;
      if (this.cmdStatus.responses.length > 0)
        this.cmdStatus.respModal = true;
    }
  },
  mounted() {
    this.username = sessionStorage.getItem('rttys-username') || '';

    this.axios.get('/isadmin').then(res => {
      this.isadmin = res.data.admin;
    });

    this.getDevices();
  }
}
</script>
