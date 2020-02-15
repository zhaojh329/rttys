import Vue from 'vue'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import './plugins/element.js'
import axios from 'axios'
import VueAxios from 'vue-axios'
import Contextmenu from '@/components/contextmenu'

Vue.config.productionTip = false;

Vue.use(VueAxios, axios);

Vue.component('Contextmenu', Contextmenu);

new Vue({
  router,
  i18n,
  render: h => h(App)
}).$mount('#app');
