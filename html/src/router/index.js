import Vue from 'vue'
import Router from 'vue-router'
import Login from '@/components/Login'
import Home from '@/components/Home'
import Rtty from '@/components/Rtty'

Vue.use(Router)

export default new Router({
	routes: [
		{
			path: '/login',
			name: 'Login',
			component: Login
		},
		{
			path: '/',
			name: 'Home',
			component: Home
		},
		{
			path: '/rtty',
			name: 'Rtty',
			component: Rtty
		}
	]
})
