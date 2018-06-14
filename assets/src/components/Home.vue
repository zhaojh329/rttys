<template>
	<div id="home">
         <Row type="flex" align="bottom">
            <Col span="6">
                <Input v-model="filterString" icon="search" size="large" @on-change="handleSearch" :placeholder="$t('Please enter the filter key...')" style="width: 400px" />
            </Col>
            <Col span="3" offset="15" class="counter">{{ $t('Online Device: {count}', {count: devlists.length}) }}</Col>
         </Row>
        <Table :loading="loading" :columns="devlistTitle" :data="filtered" style="margin-top: 10px; width: 100%" :no-data-text="$t('No devices connected')"></Table>
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
			devlistTitle: [
                {
                    title: 'ID',
                    key: 'id',
                    sortType: 'asc',
                    sortable: true
                }, {
                    title: this.$t('Uptime'),
                    key: 'uptime',
                    sortable: true,
                    render: (h, params) => {
                        return h('span', '%t'.format(params.row.uptime));
                    }
                }, {
                    title: this.$t('Description'),
                    key: 'description'
                }, {
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
            ]
		}
	},
	methods: {
		handleSearch() {
            this.filtered = this.devlists.filter(d => {
                return d.id.indexOf(this.filterString) > -1 || d.description.indexOf(this.filterString) > -1;
            });
        },
        getDevices() {
            this.$http.get('/devs').then(res => {
                this.loading = false;
                this.devlists = res.data;
                this.handleSearch();
            }).catch(err => {
                this.$router.push('/login');
            });
        }
	},
	mounted() {
        if (this.$root.$data.interval)
            clearInterval(this.$root.$data.interval);

        this.$root.$data.interval = setInterval(()=> {
            this.getDevices();
        }, 3000);

        this.getDevices();
	}
}
</script>

<style>
	#home {
		padding:10px;
	}
    .counter {
        color: #3399ff;
        font-size: 16px;
    }
</style>