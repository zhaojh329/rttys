import Vue from 'vue'
import Router from 'vue-router'
import Login from '@/components/Login'
import Home from '@/components/Home'

Vue.use(Router)

export default new Router({
    routes: [
        {path: '/', component: Login},
        {path: '/home', component: Home}
    ]
})
