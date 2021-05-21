import { useState } from 'react';

type Method = 'GET' | 'POST';

const call = async (method: Method, endpoint: string, body?: any) => {
  const res = await fetch(endpoint, { method, body: JSON.stringify(body) });
  const contentType = res.headers.get('content-type');

  if (contentType?.includes('application/json')) {
    return res.json();
  }

  if (contentType?.includes('text/html')) {
    return res.text();
  }

  return res;
};

export default function Sandbox() {
  const [method, setMethod] = useState<Method>('GET');
  const [endpoint, setEndpoint] = useState('/api');
  const [body, setBody] = useState('');
  const [data, setData] = useState(null);

  const handleSend = async () => {
    const data = await call(
      method,
      endpoint,
      body.trim() ? JSON.parse(body.trim()) : undefined
    );
    setData(data);
  };

  return (
    <div className='sandbox'>
      <form onSubmit={(e) => e.preventDefault()}>
        <select
          value={method}
          onChange={(e) => setMethod(e.target.value as Method)}
        >
          <option value='GET'>GET</option>
          <option value='POST'>POST</option>
        </select>

        <input
          type='text'
          placeholder='endpoint'
          value={endpoint}
          onChange={(e) => setEndpoint(e.target.value)}
        />

        <br />

        <textarea
          value={body}
          placeholder='body (must be valid JSON)'
          onChange={(e) => setBody(e.target.value)}
        ></textarea>

        <br />

        <button onClick={handleSend}>Send</button>
      </form>

      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}
