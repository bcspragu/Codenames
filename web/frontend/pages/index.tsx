import Link from 'next/link';

export default function Home() {
  return (
    <div>
      <h1>codenames.ai</h1>
      Go to{' '}
      <Link href='/sandbox'>
        <a>/sandbox</a>
      </Link>{' '}
      or{' '}
      <Link href='/game/someBogusId'>
        <a>/game/[whateverId]</a>
      </Link>{' '}
      to explore
    </div>
  );
}
