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

Vue.config.productionTip = false

Vue.use(VueI18n);
Vue.use(iView);

Vue.use(VueContextMenu);

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

/* eslint-disable no-new */
new Vue({
	i18n: i18n,
    el: '#app',
    render: (h)=>h(App)
});
