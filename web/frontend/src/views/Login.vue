<template>
  <div class="container">
    <div class="columns is-mobile is-centered is-gapless">
      <div class="column is-11-mobile is-8-desktop">
        <b-field grouped>
          <b-input placeholder="Enter a username..." expanded v-model="username"></b-input>
          <p class="control">
            <button @click="login" class="button is-primary">Join</button>
          </p>
        </b-field>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

@Component
export default class Login extends Vue {
  private username: string = '';

  private login(): void {
    this.axios.post('/api/user', {name: this.username}).then((resp) => {
      const data = resp.data;
      if (!data.CookieValue) {
        return;
      }
      let expires = '';
      const date = new Date();

      // Save cookie for a year.
      date.setTime(date.getTime() + (365 * 24 * 60 * 60 * 1000));
      expires = '; expires=' + date.toUTCString();
      document.cookie = 'auth=' + data.CookieValue + expires + '; path=/';

      this.$root.$data.loadUser().then(() => {
        this.$router.push({ name: 'home' });
      });
    });
  }
}
</script>

<style scoped lang="scss">
.container {
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
</style>
