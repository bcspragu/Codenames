export type Method = 'GET' | 'POST' | 'PATCH';

export const call = async (method: Method, endpoint: string, body?: any) => {
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
