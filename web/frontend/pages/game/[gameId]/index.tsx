import styles from './index.module.scss';

import { useRouter } from 'next/router';
import { useState } from 'react';
import cn from 'classnames';

const dummyWordSet: CardSet = [
  [
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
  ],
  [
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
  ],
  [
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
  ],
  [
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
  ],
  [
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
    {
      word: 'foo',
      team: 'blue',
      revealed: false,
    },
  ],
];

type Card = {
  word: string;
  team: 'red' | 'blue' | 'neutral' | 'death';
  revealed: boolean;
};

type CardRow = [Card, Card, Card, Card, Card];

type CardSet = [CardRow, CardRow, CardRow, CardRow, CardRow];

export default function Game() {
  const router = useRouter();
  const { gameId } = router.query;
  const [cardSet, setCardSet] = useState<CardSet | null>(dummyWordSet);

  console.log(cardSet);

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
      {renderGameBoard()}
    </div>
  );
}
