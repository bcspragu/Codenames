<template>
  <div id="app">
    <div>{{ banner }}</div>
    <router-view/>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import store from '@/store';
import { User } from '@/state';

@Component
export default class App extends Vue {
  private user?: User = store.state.user;

  private created(): void {
    store.loadUser();
  }

  get banner(): string {
    if (this.user) {
      return `Logged in as ${this.user.Name}`;
    }
    return 'Not logged in';
  }
}
</script>

<style lang="scss">
// Import Bulma's core
@import "~bulma/sass/utilities/_all";

// Custom styles go here
html, body, #app {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
}

// Import Bulma and Buefy styles
@import "~bulma";
@import "~buefy/src/scss/buefy";
</style>
