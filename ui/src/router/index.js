/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

import { createWebHistory, createRouter } from 'vue-router'
import axios from 'axios'

import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import Rtty from '../views/Rtty.vue'
import Error from '../views/Error.vue'

const routes = [
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
    path: '/rtty/:devid',
    name: 'Rtty',
    component: Rtty,
    props: true
  },
  {
    path: '/error/:err',
    name: 'Error',
    component: Error,
    props: true
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  if (to.path !== '/login') {
    axios.get('/alive').then(() => {
      next()
    }).catch(() => {
      next({ name: 'Login' })
    })
  } else {
    next()
  }
})

export default router
