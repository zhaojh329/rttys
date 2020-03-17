import Vue from 'vue'
import VueRouter from 'vue-router'
import Login from '../views/Login.vue'
import Home from '../views/Home.vue'
import Rtty from '../views/Rtty.vue'

Vue.use(VueRouter)

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: Login
  },
  {
    path: '/',
    name: 'home',
    component: Home
  },
  {
    path: '/rtty/:devid',
    name: 'Rtty',
    component: Rtty,
    props: true
  }
];

const router = new VueRouter({
  mode: 'history',
  routes
});

router.beforeEach((to, from, next) => {
  if (to.matched.length > 0 && to.matched[0].path === '/rtty/:devid') {
    const devid = to.params['devid'];
    Vue.axios.get(`/authorized/${devid}`).then(r => {
      if (r.data.authorized)
        next();
      else
        router.push('/login');
    });
    return;
  }

  if (to.path !== '/login' && !sessionStorage.getItem('rtty-sid')) {
    router.push('/login');
    return;
  }

  next();
});

export default router
