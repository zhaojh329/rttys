/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

import { createApp } from 'vue'
import VueClipboard from 'vue-clipboard2'
import VueAxios from 'vue-axios'
import axios from 'axios'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import ElementPlus from './element-plus'

const app = createApp(App)

app.use(VueClipboard)
app.use(VueAxios, axios)
app.use(router)
app.use(i18n)
app.use(ElementPlus)

app.mount('#app')
