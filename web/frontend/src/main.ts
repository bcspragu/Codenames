import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import './registerServiceWorker';

import axios from 'axios';
import VueAxios from 'vue-axios';

import Buefy from 'buefy';

Vue.use(VueAxios, axios);
Vue.use(Buefy);
Vue.prototype.$log = console.log.bind(console);

Vue.config.productionTip = false;

new Vue({
  data: store,
  router,
  render: (h) => h(App),
}).$mount('#app');
