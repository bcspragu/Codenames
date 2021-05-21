import { useEffect, useState } from 'react';
import '../styles/globals.css';

const call = (method: 'GET' | 'POST', endpoint: string, body?: any) => {
  return fetch(endpoint, { method, body: JSON.stringify(body) });
};

function MyApp({ Component, pageProps }) {
  const [data, setData] = useState(null);

  useEffect(() => {
    const ping = async () => {
      const res = await call('POST', '/api/user', { name: 'what' });

      const contentType = res.headers.get('content-type');

      const data = await (() => {
        if (contentType?.includes('application/json')) {
          return res.json();
        }
        if (contentType?.includes('text/html')) {
          return res.text();
        }
        return res;
      })();

      console.log('data', data);

      setData(data);
    };

    ping();
  }, [null]);

  return <Component {...pageProps} />;
}

export default MyApp;
