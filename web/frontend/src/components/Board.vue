<template>
  <div>
    <div v-for="row in board.Cards">
      <span v-for="cell in row"> |{{cell.Codename}}| </span>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import { GameBoard } from '@/views/Home.vue';

@Component
export default class Board extends Vue {
  @Prop() private board!: GameBoard;
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
</style>
