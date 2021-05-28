import Link from 'next/link';
import { useEffect, useState } from 'react';
import { call } from '../utils/call';

export default function Home() {
  const [name, setName] = useState('');
  const [wasLoggedIn, setWasLoggedIn] = useState(false);

  useEffect(() => {
    const initState = async () => {
      const res = await call('GET', '/api/user');
      console.log(res);

      if (res && res.name) {
        setName(res.name);
        setWasLoggedIn(true);
      }
    };

    initState();
  }, [null]);

  const handleCreateNewGame = async () => {
    if (wasLoggedIn) {
      await call('PATCH', '/api/user', { name: name.trim() });
    } else {
      await call('POST', '/api/user', { name: name.trim() });
    }

    const res = await call('POST', '/api/game');

    console.log(res); // TODO put the gameId in store, ideally
  };

  return (
    <div>
      <h1>codenames.ai</h1>
      <div className='tile'>
        <form action='submit' onSubmit={(e) => e.preventDefault()}>
          <input
            type='text'
            placeholder='Name'
            value={name}
            onChange={(e) => setName(e.target.value.trimStart())}
          />
          <button type='submit' onClick={handleCreateNewGame}>
            Create new game
          </button>
        </form>
      </div>
    </div>
  );
}
