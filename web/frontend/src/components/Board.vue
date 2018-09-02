<template>
  <div class="container">
    <div class="columns is-mobile is-centered is-gapless">
      <div class="column is-12-mobile is-8-desktop">
        <div class="row columns is-mobile is-gapless is-centered" v-for="row in board.Cards">
          <div class="column cell" v-for="cell in row">
            <div class="body is-size-6 is-size-7-mobile" :class="color(cell)">{{format(cell.Codename)}}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import { GameBoard, Card } from '@/views/Home.vue';

@Component
export default class Board extends Vue {
  @Prop() private board!: GameBoard;

  private format(cn: string): string {
    return cn.split('_').map((c) => c.charAt(0).toUpperCase() + c.substr(1)).join(' ');
  }

  private color(cd: Card): string {
    switch (cd.Agent) {
      case 1:
        return 'red';
      case 2:
        return 'blue';
      case 4:
        return 'grey';
      default:
        return '';
    }
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

.row.columns {
  margin-bottom: 0.6rem;
}

.cell {
  >.body {
    margin-left: 0.3rem;
    margin-right: 0.3rem;
    min-height: 3rem;

    text-overflow: ellipsis;
    whitespace: nowrap;
    overflow: hidden;

    font-weight: bold;
    display: flex;
    justify-content: center;
    align-items: center;
    border-radius: 0.25rem;

    &.red {
      background-color: rgba(255, 0, 0, 0.3);
    }

    &.blue {
      background-color: rgba(0, 0, 255, 0.3);
    }

    &.grey {
      background-color: rgba(0, 0, 0, 0.3);
    }
  }
}

</style>
