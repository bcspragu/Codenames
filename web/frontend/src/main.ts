import Vue from 'vue';
import App from './App.vue';
import router from './router';
import './registerServiceWorker';

import axios from 'axios';
import VueAxios from 'vue-axios';
Vue.use(VueAxios, axios);
Vue.prototype.$log = console.log.bind(console);

Vue.config.productionTip = false;

new Vue({
  router,
  render: (h) => h(App),
}).$mount('#app');
