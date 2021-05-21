import Vue from 'vue';
import Router from 'vue-router';
import { NavigationGuard, Route } from 'vue-router';
import store from '@/store';

import Home from '@/views/Home.vue';
import Game from '@/views/Game.vue';
import Login from '@/views/Login.vue';

Vue.use(Router);

const authCheck: NavigationGuard = (to: Route, from: Route, next: any) => {
  store.getUser().then((user) => {
    if (user) {
      next();
      return;
    }
    next({path: 'login'});
  });
};

export default new Router({
  routes: [
    {
      path: '/login',
      name: 'login',
      component: Login,
    },
    {
      path: '/',
      name: 'home',
      component: Home,
      beforeEnter: authCheck,
    },
    {
      path: '/game/:id',
      name: 'game',
      component: Game,
      beforeEnter: authCheck,
    },
  ],
});
