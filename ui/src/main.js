import Vue from 'vue'
import App from './App.vue'
import router from './router'
import axios from 'axios'
import VueAxios from 'vue-axios'
import VueClipboard from 'vue-clipboard2'
import i18n from './plugins/vue-i18n'
import './plugins/view-design'
import './assets/iconfont/iconfont.css'

Vue.config.productionTip = false

Vue.use(VueClipboard)
Vue.use(VueAxios, axios);

Vue.prototype.$axios = axios

// 进入后配置一次
Vue.prototype.BASE_URL = getConfigItem('BASE_URL_PROD');
console.log('接口配置的基础地址1', Vue.prototype.BASE_URL)

axios.defaults.baseURL = Vue.prototype.BASE_URL;

new Vue({
  router,
  i18n,
  render: h => h(App)
}).$mount('#app')
