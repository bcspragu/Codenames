import styles from './index.module.scss';

import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import cn from 'classnames';

/**
 * Notes
 *
 * Team assignments should not be visible anywhere client side except to spymasters or after a card has been clicked/revealed.
 * Server should have assignments stored (by id)
 */

type Team = 'red' | 'blue' | 'neutral' | 'death';

type Card = {
  word: string;
  team: Team;
  revealed: boolean;
};

type CardRow = [Card, Card, Card, Card, Card];

type CardSet = [CardRow, CardRow, CardRow, CardRow, CardRow];

// Stolen from https://stackoverflow.com/questions/2450954/how-to-randomize-shuffle-a-javascript-array
function shuffleArray(array) {
  for (var i = array.length - 1; i > 0; i--) {
    var j = Math.floor(Math.random() * (i + 1));
    var temp = array[i];
    array[i] = array[j];
    array[j] = temp;
  }
}

const getAssignCardSetClientSideWhichMakesNoSense = (
  words: string[],
  firstTeam: 'red' | 'blue'
): CardSet => {
  const secondTeam = firstTeam === 'red' ? 'blue' : 'red';

  const deck = words.map((word, i) => {
    const team: Team = (() => {
      // First team to go gets 9 cards, second gets 8
      if (i <= 8) {
        return firstTeam;
      }

      if (i >= 9 && i <= 16) {
        return secondTeam;
      }

      if (i === 17) {
        return 'death';
      }

      return 'neutral';
    })();

    return {
      word,
      team,
      revealed: false,
    };
  });

  shuffleArray(deck);

  return [
    // @ts-ignore
    deck.slice(0, 5),
    // @ts-ignore
    deck.slice(5, 10),
    // @ts-ignore
    deck.slice(10, 15),
    // @ts-ignore
    deck.slice(15, 20),
    // @ts-ignore
    deck.slice(20),
  ];
};

export default function Game() {
  const router = useRouter();
  const { gameId } = router.query;
  const [cardSet, setCardSet] = useState<CardSet | null>();
  const [firstTeam, setFirstTeam] = useState<'red' | 'blue'>('red');

  useEffect(() => {
    const setInitialState = async () => {
      const randomWords = await (
        await fetch('https://random-word-api.herokuapp.com/word?number=25')
      ).json();

      const firstTeam = Math.random() < 0.5 ? 'red' : 'blue';

      setCardSet(
        getAssignCardSetClientSideWhichMakesNoSense(randomWords, firstTeam)
      );
      setFirstTeam(firstTeam);
    };

    setInitialState();
  }, [null]);

  const renderGameBoard = () => {
    if (!cardSet) {
      return 'Loading';
    }

    const renderCard = (card: Card, rowIndex: number, cardIndex: number) => {
      const handleClick = () => {
        // @ts-ignore
        setCardSet((prevSet) => {
          return [
            ...prevSet.slice(0, rowIndex),
            [
              ...prevSet[rowIndex].slice(0, cardIndex),
              { ...prevSet[rowIndex][cardIndex], revealed: true },
              ...prevSet[rowIndex].slice(cardIndex + 1),
            ],
            ...prevSet.slice(rowIndex + 1),
          ];
        });
      };

      return (
        <div
          key={cardIndex}
          className={cn(
            styles.gameBoardCard,
            card.revealed && {
              [styles.revealed]: true,
              [styles.red]: card.team === 'red',
              [styles.blue]: card.team === 'blue',
              [styles.neutral]: card.team === 'neutral',
              [styles.death]: card.team === 'death',
            }
          )}
          onClick={handleClick}
          tabIndex={0}
        >
          {card.word}
        </div>
      );
    };

    const renderCardRow = (row: CardRow, rowIndex: number) => {
      return (
        <div key={rowIndex} className={styles.gameBoardRow}>
          {row.map((card, cardIndex) => renderCard(card, rowIndex, cardIndex))}
        </div>
      );
    };

    return <div className='game-board'>{cardSet.map(renderCardRow)}</div>;
  };

  return (
    <div className='game-page'>
      <div>Game id: {gameId}</div>
      {cardSet && <div>First team (9 cards): {firstTeam}</div>}
      {renderGameBoard()}
    </div>
  );
}
