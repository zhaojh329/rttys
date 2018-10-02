import Vue from 'vue'
import './plugins/axios'
import App from './App.vue'
import router from './router'
import './plugins/iview.js'
import i18n from './rtty-i18n'
import 'string-format-easy'

Vue.config.productionTip = false

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

new Vue({
    i18n,
    router,
    render: h => h(App)
}).$mount('#app')
