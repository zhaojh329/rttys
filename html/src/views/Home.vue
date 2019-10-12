<template>
    <div style="padding:10px">
        <Button style="margin-right: 4px;" type="primary" shape="circle" icon="md-refresh" @click="handleRefresh" :disabled="loading">{{$t('Refresh List')}}</Button>
        <Input style="margin-right: 4px;width:200px" v-model="filterString" icon="search" size="large" @on-change="handleSearch" :placeholder="$t('Please enter the filter key...')" />
        <Button style="margin-right: 4px;" @click="showCmdForm" type="primary" :disabled="cmdStatus.execing > 0"><Icon type="search"/>{{$t('executive command')}}</Button>
        <div class="counter">
            {{ $t('device-count', {count: devlists.length}) }}
        </div>
        <Table :height="tableHeight" :loading="loading" :columns="devlistTitle" :data="filtered" style="margin-top: 10px; width: 100%" :no-data-text="$t('No devices connected')" @on-selection-change='handleSelection'>
            <template slot-scope="{ row }" slot="uptime">
                <span>{{ '%t'.format(row.uptime) }}</span>
            </template>
            <template slot-scope="{ row }" slot="action">
                <Button type="primary" @click="connectDevice(row.id)">{{$t('Connect')}}</Button>
            </template>
        </Table>
        <Modal v-model="cmdModal" :title="$t('executive command')">
            <Form :model="cmdData" ref="cmdForm" :rules="cmdRuleValidate" :label-width="80">
                <FormItem :label="$t('Username')" prop="username">
                    <Input v-model="cmdData.username"></Input>
                </FormItem>
                <FormItem :label="$t('Password')" prop="password">
                    <Input type="password" v-model="cmdData.password"></Input>
                </FormItem>
                <FormItem :label="$t('Command')" prop="cmd">
                    <Input v-model="cmdData.cmd"></Input>
                </FormItem>
                <FormItem :label="$t('Parameter')" prop="params">
                    <Tag v-for="(item, index) in cmdData.params" :key="item + index" closable @on-close="handleDelCmdParam(index)" :fade="false">{{ item }}</Tag>
                    <Input v-model="cmdData.currentParam" icon="md-add-circle" :placeholder="$t('Please enter a single parameter')"  @on-click="handleAddCmdParam" @on-keyup.enter="handleAddCmdParam" />
                </FormItem>
                <FormItem :label="$t('Environment variable')" prop="env">
                    <Tag v-for="(v, k) in cmdData.env" :key="v + k" closable @on-close="handleDelCmdEnv(k)" :fade="false">{{ k + '=' + v }}</Tag>
                    <Input v-model="cmdData.currentEnv" icon="md-add-circle" :placeholder="$t('Please enter a single environment')"  @on-click="handleAddCmdEnv" @on-keyup.enter="handleAddCmdEnv" />
                </FormItem>
            </Form>
            <div slot="footer">
                <Button type="primary" @click="doCmd">{{$t('OK')}}</Button>
                <Button @click="cmdModal = false" style="margin-left: 8px">{{$t('Cancel')}}</Button>
            </div>
        </Modal>
        <Modal v-model="cmdStatus.modal" :title="$t('status of executive command')" :closable="false" :mask-closable="false">
            <Progress :percent="cmdStatusPercent" status="active"></Progress>
            <p>{{ $t('cmd-status-total', {count: cmdStatus.total}) }}</p>
            <p>{{ $t('cmd-status-fail', {count: cmdStatus.fail}) }}</p>
            <div slot="footer">
                <Button type="primary" size="large" :disabled="cmdStatus.execing > 0" @click="showCmdResp">{{$t('OK')}}</Button>
                <Button type="error" size="large" :disabled="cmdStatus.execing == 0" @click="ignoreCmdResp">{{$t('Ignore')}}</Button>
            </div>
        </Modal>
        <Modal v-model="cmdStatus.respModal" :title="$t('Response of executive command')" :width="1000">
            <Table :columns="cmdStatus.response.columns" :data="cmdStatus.response.data" height="300" :no-data-text="$t('No Response')"></Table>
            <div slot="footer"></div>
        </Modal>
    </div>
</template>

<script>

export default {
    name: 'Home',
    data() {
        return {
            filterString: '',
            loading: true,
            devlists: [],
            filtered: [],
            selection: [],
            devlistTitle: [
                {
                    type: 'index',
                    width: 80
                },
                {
                    type: 'selection',
                    width: 60,
                    align: 'center'
                },
                {
                    title: this.$t('Device ID'),
                    key: 'id',
                    sortType: 'asc',
                    sortable: true
                },
                {
                    title: this.$t('Uptime'),
                    key: 'uptime',
                    sortable: true,
                    slot: 'uptime'
                },
                {
                    title: this.$t('Description'),
                    key: 'description'
                },
                {
                    width: 150,
                    align: 'center',
                    slot: 'action'
                }
            ],
            cmdModal: false,
            cmdStatus: {
                modal: false,
                execing: 0,
                fail: 0,
                running: {},
                respModal: false,
                response: {
                    columns: [
                        {
                            type: 'index',
                            width: 60,
                            align: 'center'
                        },
                        {
                            title: this.$t('Device ID'),
                            key: 'devid'
                        },
                        {
                            title: this.$t('Command'),
                            key: 'cmd'
                        },
                        {
                            title: this.$t('Error Code'),
                            key: 'err'
                        },
                        {
                            title: this.$t('Error Message'),
                            key: 'msg'
                        },
                        {
                            title: this.$t('Status Code'),
                            key: 'code'
                        },
                        {
                            title: this.$t('Stdout'),
                            key: 'stdout'
                        },
                        {
                            title: this.$t('Stderr'),
                            key: 'stderr'
                        }
                    ],
                    data: []
                }
            },
            cmdData: {
                username: '',
                password: '',
                cmd: '',
                params: [],
                currentParam: '',
                env: {},
                currentEnv: ''
            },
            cmdRuleValidate: {
                username: [
                    { required: true, trigger: 'blur', message: this.$t('username is required') }
                ],
                cmd: [
                    { required: true, trigger: 'blur', message: this.$t('command is required') }
                ]
            },
            tableHeight: 0
        }
    },
    methods: {
        handleSearch() {
            this.filtered = this.devlists.filter(d => {
                return d.id.indexOf(this.filterString) > -1 || d.description.indexOf(this.filterString) > -1;
            });
        },
        getDevices() {
            this.$axios.get(process.env.BASE_URL+'devs').then(res => {
                this.loading = false;
                this.devlists = res.data;
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
        connectDevice(devid) {
            this.$router.push({path: '/rtty', query: {devid: devid}});
        },
        showCmdForm() {
            if (this.selection.length < 1) {
                this.$Message.error(this.$t('Please select the devices you want to operate.'));
                return;
            }
            this.cmdModal = true;
        },
        queryCmdResp() {
            let count = 0;

            for (let token in this.cmdStatus.running) {
                let item = this.cmdStatus.running[token];

                if (item.querying)
                    continue;

                item.querying = true;

                this.$axios.get(process.env.BASE_URL+'cmd?token=' + token).then((response) => {
                    let resp = response.data;

                    if (resp.err == 1005) {
                        item.querying = false;
                        return;
                    }

                    if (resp.err && resp.err != 0)
                        this.cmdStatus.fail++;

                    this.cmdStatus.execing--;

                    this.cmdStatus.response.data.push({
                        devid: item.devid,
                        cmd: item.cmd,
                        code: resp.code,
                        err: resp.err,
                        msg: resp.msg,
                        stdout: resp.stdout && window.atob(resp.stdout),
                        stderr: resp.stderr && window.atob(resp.stderr)
                    });

                    delete this.cmdStatus.running[token];
                });

                count++;

                if (count > 10)
                    break;
            }

            if (this.cmdStatus.execing > 0)
                setTimeout(this.queryCmdResp, 500);
        },
        handleDelCmdParam(index) {
            this.cmdData.params.splice(index, 1);
        },
        handleAddCmdParam() {
            this.cmdData.currentParam = this.cmdData.currentParam.trim();
            if (this.cmdData.currentParam != '') {
                this.cmdData.params.push(this.cmdData.currentParam);
                this.cmdData.currentParam = '';
            }
        },
        handleDelCmdEnv(key) {
            this.$delete(this.cmdData.env, key);
        },
        handleAddCmdEnv() {
            this.cmdData.currentEnv = this.cmdData.currentEnv.trim();
            if (this.cmdData.currentEnv != '') {
                let e = this.cmdData.currentEnv.split('=');
                if (e.length == 2)
                    this.$set(this.cmdData.env, [e[0]], e[1]);
                this.cmdData.currentEnv = '';
            }
        },
        doCmd() {
            this.$refs['cmdForm'].validate((valid) => {
                if (valid) {
                    this.cmdModal = false;
                    this.cmdStatus.modal = true;
                    this.cmdStatus.total = this.selection.length;
                    this.cmdStatus.execing = this.selection.length;
                    this.cmdStatus.fail = 0;
                    this.cmdStatus.running = {};
                    this.cmdStatus.response.data = [];

                    this.selection.forEach((item) => {
                        let data = {
                            devid: item.id,
                            username: this.cmdData.username,
                            password: this.cmdData.password,
                            cmd: this.cmdData.cmd.trim(),
                            params: this.cmdData.params,
                            env: this.cmdData.env
                        };

                        this.$axios.post(process.env.BASE_URL+'cmd', data).then((response) => {
                            let resp = response.data;

                            if (resp.token) {
                                this.cmdStatus.running[resp.token] = {
                                    devid: item.id,
                                    cmd: data.cmd
                                };
                                return;
                            }

                            this.cmdStatus.execing--;
                            this.cmdStatus.fail++;

                            this.cmdStatus.response.data.push({
                                devid: item.id,
                                cmd: data.cmd,
                                err: resp.err,
                                msg: resp.msg
                            });
                        });
                    });

                    setTimeout(this.queryCmdResp, 100);
                }
            });
        },
        ignoreCmdResp() {
            this.cmdStatus.execing = 0;
            this.cmdStatus.running = {};

            this.cmdStatus.respModal = true;
            this.cmdStatus.modal = false;
        },
        showCmdResp() {
            this.cmdStatus.modal = false;
            if (this.cmdStatus.response.data.length > 0)
                this.cmdStatus.respModal = true;
        }
    },
    computed: {
        cmdStatusPercent() {
            let percent = (this.cmdStatus.total - this.cmdStatus.execing) / this.cmdStatus.total * 100;
            return parseInt(percent);
        }
    },
    mounted() {
        this.getDevices();

        this.tableHeight = document.body.clientHeight - 70;
        window.addEventListener('resize', () => {
            this.tableHeight = document.body.clientHeight - 70;
        });
    }
};
</script>

<style>
.counter {
    float: right;
    color: #3399ff;
    font-size: 16px;
}
</style>
