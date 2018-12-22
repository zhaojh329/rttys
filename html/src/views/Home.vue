<template>
    <div id="home">
        <Button style="margin-right: 4px;" type="primary" shape="circle" icon="md-refresh" @click="handleRefresh" :disabled="loading">{{$t('Refresh List')}}</Button>
        <Input style="margin-right: 4px;width:200px" v-model="filterString" icon="search" size="large" @on-change="handleSearch" :placeholder="$t('Please enter the filter key...')" />
        <Button style="margin-right: 4px;" @click="showCmdForm" type="primary" :disabled="cmdStatus.execing > 0"><Icon type="search"/>{{$t('executive command')}}</Button>
        <div class="counter">
            {{ $t('device-count', {count: devlists.length}) }}
        </div>
        <Table :loading="loading" :columns="devlistTitle" :data="filtered" style="margin-top: 10px; width: 100%" :no-data-text="$t('No devices connected')" @on-selection-change='handleSelection'></Table>
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
                    <Input v-model="cmdData.params"></Input>
                </FormItem>
                <FormItem :label="$t('Environment variable')" prop="env">
                    <Input v-model="cmdData.env"></Input>
                </FormItem>
            </Form>
            <div slot="footer">
                <Button type="primary" @click="doCmd">{{$t('OK')}}</Button>
                <Button @click="cmdModal = false" style="margin-left: 8px">{{$t('Cancel')}}</Button>
            </div>
        </Modal>
        <Modal v-model="cmdStatus.modal" :title="$t('status of executive command')" :closable="false" :mask-closable="false">
            <div v-if="cmdStatus.total > 1">
                <Progress :percent="(cmdStatus.total - cmdStatus.execing) / cmdStatus.total * 100" status="active"><span>100%</span></Progress>
                <p>{{ $t('cmd-status-total', {count: cmdStatus.total}) }}</p>
                <p>{{ $t('cmd-status-succeed', {count: cmdStatus.succeed}) }}</p>
                <p>{{ $t('cmd-status-fail', {count: cmdStatus.fail}) }}</p>
            </div>
            <div v-else>
                <p v-if="cmdStatus.err > 0">{{cmdStatus.msg}}</p>
                <div v-else>
                    <p>Code: {{cmdStatus.code}}</p>
                    <Divider />
                    <span>Stdout:</span>
                    <Input v-model="cmdStatus.stdout" type="textarea" readonly />
                    <Divider />
                    <span>Stderr:</span>
                    <Input v-model="cmdStatus.stderr" type="textarea" readonly />
                </div>
            </div>
            <div slot="footer">
                <Button type="primary" size="large" long :disabled="cmdStatus.execing > 0" @click="cmdStatus.modal = false">{{$t('OK')}}</Button>
            </div>
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
                    type: 'selection',
                    width: 60,
                    align: 'center'
                },
                {
                    title: 'ID',
                    key: 'id',
                    sortType: 'asc',
                    sortable: true
                },
                {
                    title: this.$t('Uptime'),
                    key: 'uptime',
                    sortable: true,
                    render: (h, params) => {
                        return h('span', '%t'.format(params.row.uptime));
                    }
                },
                {
                    title: this.$t('Description'),
                    key: 'description'
                },
                {
                    width: 150,
                    align: 'center',
                    render: (h, params) => {
                        return h('Button', {
                            props: { type: 'primary' },
                            on: {
                                click: () => {
                                    this.$router.push({path: '/rtty', query: {devid: params.row.id}});
                                }
                            }
                        }, this.$t('Connect'));
                    }
                }
            ],
            cmdModal: false,
            cmdStatus: {
                modal: false,
                execing: 0,
                succeed: 0,
                fail: 0,
                stdout: '',
                stderr: '',
                code: 0
            },
            cmdData: {
                username: '',
                password: '',
                cmd: '',
                params: '',
                env: ''
            },
            cmdRuleValidate: {
                username: [
                    { required: true, trigger: 'blur', message: this.$t('username is required') }
                ],
                cmd: [
                    { required: true, trigger: 'blur', message: this.$t('command is required') }
                ]
            }
        }
    },
    methods: {
        handleSearch() {
            this.filtered = this.devlists.filter(d => {
                return d.id.indexOf(this.filterString) > -1 || d.description.indexOf(this.filterString) > -1;
            });
        },
        getDevices() {
            this.$axios.get('/devs').then(res => {
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
        showCmdForm() {
            if (this.selection.length < 1) {
                this.$Message.error(this.$t('Please select the devices you want to operate.'));
                return;
            }
            this.cmdModal = true;
        },
        doCmd() {
            this.$refs['cmdForm'].validate((valid) => {
                if (valid) {
                    this.cmdModal = false;
                    this.cmdStatus.modal = true;
                    this.cmdStatus.total = this.selection.length;
                    this.cmdStatus.execing = this.selection.length;
                    this.cmdStatus.succeed = 0;
                    this.cmdStatus.fail = 0;
                    this.cmdStatus.code = 0;
                    this.cmdStatus.stdout = '';
                    this.cmdStatus.stderr = '';
                    this.cmdStatus.err = 0;
                    this.cmdStatus.msg = '';

                    this.selection.forEach((item) => {
                        let data = {
                            devid: item.id,
                            username: this.cmdData.username,
                            password: this.cmdData.password,
                            cmd: this.cmdData.cmd.trim(),
                            params: [],
                            env: {}
                        };

                        this.cmdData.params = this.cmdData.params.trim();
                        if (this.cmdData.params != '')
                            data.params = this.cmdData.params.split(' ');

                        this.cmdData.env = this.cmdData.env.trim();
                        if (this.cmdData.env != '') {
                            this.cmdData.env.split(' ').forEach((item) => {
                                let e = item.split('=');
                                if (e.length == 2)
                                    data.env[e[0]] = e[1];
                            });
                        }

                        this.$axios.post('/cmd', JSON.stringify(data)).then((response) => {
                            let cmdresp = response.data;
                            if (typeof cmdresp == 'string') {
                                cmdresp = cmdresp.replace(/\n/g, '\\n').replace(/\r/g, '\\r');
                                cmdresp = JSON.parse(cmdresp);

                                if (this.cmdStatus.total == 1)
                                    cmdresp.stdout = cmdresp.stdout.replace(/\\n/g, '\n').replace(/\\r/g, '\r');
                            }

                            this.cmdStatus.execing--;
                            if (cmdresp.err)
                                this.cmdStatus.fail++;
                            else
                                this.cmdStatus.succeed++;

                            if (this.cmdStatus.total == 1) {
                                this.cmdStatus.err = cmdresp.err || 0;
                                this.cmdStatus.msg = cmdresp.msg || '';
                                this.cmdStatus.code = cmdresp.code;
                                this.cmdStatus.stdout = cmdresp.stdout;
                                this.cmdStatus.stderr = cmdresp.stderr;
                            }
                        });
                    });
                }
            });
        }
    },
    mounted() {
        this.getDevices();
    }
};
</script>

<style>
    #home {
        padding:10px;
    }
    .counter {
        float: right;
        color: #3399ff;
        font-size: 16px;
    }
    .cmdbtn {
        margin-left: 4px;
    }
</style>
