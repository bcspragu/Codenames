<template>
  <div class="container">
    <div class="columns is-mobile is-centered is-gapless">
      <div class="column is-12-mobile is-8-desktop">
        <Board :board="board"/>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import Board from '@/components/Board.vue'; // @ is an alias to /src

export interface Card {
  Codename: string;
  Agent: number;
  Revealed: boolean;
}

export interface GameBoard {
  Cards: Card[][];
}

@Component({
  components: {
    Board,
  },
})
export default class Home extends Vue {
  private board: GameBoard = {Cards: []};

  private created(): void {
    this.axios.get('/api/newBoard').then((resp) => {
      this.board = resp.data;
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
