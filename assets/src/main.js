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

Vue.config.productionTip = false

Vue.use(VueI18n);
Vue.use(iView);

Vue.use(VueContextMenu)

const Locales = {
	'zh-CN': {
		'Description': '描述',
		'Uptime': '在线时长',
		'Connect': '连接',
		'Please enter the filter key...': '请输入关键字进行过滤……',
		'No devices connected': '没有设备连接',
		'Upload file to device': '上传文件到设备',
		'Download file from device': '从设备下载文件',
		'Increase font size': '增大字体',
		'Decrease font size': '减小字体',
		'Select the file to upload': '选择您要上传的文件',
		'Uploading': '正在上传',
		'Click to upload': '上传',
		'Upload success': '上传成功',
		'Download Finish': '下载成功',
		'Upload canceled': '上传终止',
		'Download canceled': '下载终止',
		'Device offline': '设备离线',
		'modification': '修改时间'
	},
}

if (!navigator.language)
	navigator.language = 'en-US';

const i18n = new VueI18n({
	locale: navigator.language,
	messages: {
		'zh-CN': Object.assign(zhLocale, Locales['zh-CN']),
		'en-US': enLocale
	}
})

/* eslint-disable no-new */
new Vue({
	i18n: i18n,
    el: '#app',
    render: (h)=>h(App)
});
