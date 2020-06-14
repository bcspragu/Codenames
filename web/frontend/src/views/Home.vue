<template>
  <div class="container">
    <div class="columns is-mobile is-centered is-gapless">
      <div class="column is-11-mobile is-8-desktop">
        <button @click="newGame" class="button is-large is-fullwidth">New Game</button>
      </div>
    </div>
    <hr>
    <div class="columns is-mobile is-centered is-gapless">
      <div class="column is-11-mobile is-8-desktop">
        <div class="has-text-centered is-size-4">Pending Games</div>
        <div class="game-list columns is-mobile is-centered is-gapless">
          <div class="column is-8-mobile is-4-desktop has-text-centered">
            <div v-for="id in gameIDs">
              <router-link :to="{ name: 'game', params: { id: id }}">{{id}}</router-link>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

@Component
export default class Home extends Vue {
  private gameIDs: string[] = [];

  private created(): void {
    this.axios.get('/api/games').then((resp) => {
      this.gameIDs = resp.data;
    });
  }

  private newGame(): void {
    this.axios.post('/api/game').then((resp) => {
      this.$router.push({ name: 'game', params: { id: resp.data.ID }});
    });
  }
}
</script>

<style scoped lang="css">
.container {
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.game-list {
  margin-top: 0.75rem;
}
</style>
