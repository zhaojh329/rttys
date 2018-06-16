// The Vue build version to load with the `import` command
// (runtime-only or standalone) has been set in webpack.base.conf with an alias.
import Vue from 'vue'
import App from './App'
import iView from 'iview'
import 'iview/dist/styles/iview.css'
import VueI18n from 'vue-i18n'
import zhLocale from 'iview/dist/locale/zh-CN'
import enLocale from 'iview/dist/locale/en-US'
import 'string-format-easy'
import VueContextMenu from 'vue-contextmenu-easy'
import RttyI18n from './rtty-i18n'
import router from './router'
import axios from 'axios'
import VueAxios from 'vue-axios'

Vue.config.productionTip = false

Vue.use(VueI18n);
Vue.use(iView);

Vue.use(VueContextMenu);

Vue.use(VueAxios, axios)

const messages = {
	'zh-CN': Object.assign(zhLocale, RttyI18n['zh-CN']),
	'en-US': Object.assign(enLocale, RttyI18n['en-US'])
};

let language = navigator.language;

if (!messages[language])
	language = 'en-US';

const i18n = new VueI18n({
	locale: language,
	messages: messages
});


router.beforeEach((to, from, next) => {
	if (to.path == '/rtty' && to.query.devid) {
		next();
		return;
	}

	if (to.path == '/' && to.query.id) {
		router.push({path: '/rtty', query: {devid: to.query.id, username: to.query.username, password: to.query.password}});
		return;
	}

	if (to.path != '/login' && !sessionStorage.getItem('rtty-sid')) {
		router.push('/login');
		return;
	}

	next();
});

/* eslint-disable no-new */
new Vue({
	i18n: i18n,
    el: '#app',
    router,
    render: (h)=>h(App)
});
