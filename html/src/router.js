import Vue from 'vue'
import Router from 'vue-router'
import Login from './views/Login.vue'
import Home from './views/Home.vue'
import Rtty from './views/Rtty.vue'

Vue.use(Router)

export default new Router({
    routes: [
        {
            path: '/login',
            name: 'login',
            component: Login
        },
        {
            path: '/',
            name: 'home',
            component: Home
        },
        {
            path: '/rtty',
            name: 'rtty',
            component: Rtty
        }
    ]
})
