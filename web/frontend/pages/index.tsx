import cn from 'classnames';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useEffect, useState } from 'react';
import { call } from '../utils/call';

export default function Home() {
  const router = useRouter();
  const [view, setView] = useState<'Join' | 'Create'>('Join');
  const [name, setName] = useState('');
  const [gameId, setGameId] = useState('');
  const [wasLoggedIn, setWasLoggedIn] = useState(false);

  useEffect(() => {
    const initState = async () => {
      const res = await call('GET', '/api/user');
      console.log(res);

      if (res && res.name) {
        setWasLoggedIn(true);
        setName(res.name);
      }
    };

    initState();
  }, [null]);

  // router.query is empty on mount
  useEffect(() => {
    setGameId((router.query.gameId as string) || ''); // Fall back to an empty string, as an undefined value will trigger controlled/uncontrolled input warnings
  }, [router.query.gameId]);

  const handleSubmit = async () => {
    if (wasLoggedIn) {
      await call('PATCH', '/api/user', { name: name.trim() });
    } else {
      await call('POST', '/api/user', { name: name.trim() });
    }

    if (view === 'Join') {
      console.log('Join game called');
    } else {
      const res = await call('POST', '/api/game');
      console.log(res); // TODO put the gameId in store, ideally
    }
  };

  return (
    <div>
      <h1>codenames.ai</h1>
      <div className='tile'>
        <div>
          <div
            className={cn({ active: view === 'Join' })}
            tabIndex={0}
            onClick={() => setView('Join')}
          >
            Join game
          </div>
          <div
            className={cn({ active: view === 'Create' })}
            tabIndex={0}
            onClick={() => setView('Create')}
          >
            Create game
          </div>
        </div>

        <form action='submit' onSubmit={(e) => e.preventDefault()}>
          <input
            type='text'
            placeholder='Name'
            value={name}
            onChange={(e) => setName(e.target.value.trimStart())}
          />

          {view === 'Join' && (
            <input
              type='text'
              placeholder='Game ID'
              value={gameId}
              onChange={(e) => setGameId(e.target.value.trim())}
            />
          )}

          <div>
            <button
              type='submit'
              onClick={handleSubmit}
              disabled={!name.trim() || (view === 'Join' && !gameId)}
            >
              {view === 'Join' ? 'Join' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
