const areWebSocketsSupported = () => {
  return (
    typeof window !== 'undefined' && typeof window.WebSocket !== 'undefined'
  );
};

export class Socket {
  connection?: WebSocket;

  constructor() {
    if (!areWebSocketsSupported()) {
      alert("Your computer ain't it");
      return; // Hopefully just a server-side render...
    }

    this.connect();
  }

  connect() {
    const uri = this.getURI();

    this.connection = new WebSocket(uri);
    this.connection.onmessage = this.handleMessage;
    this.connection.onclose = this.handleClose;
  }

  getURI() {
    const { location } = window;
    const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsGameEndpoint = '/api/game/:gameId/ws';

    return `${wsProtocol}//${location.host}${wsGameEndpoint}`;
  }

  handleMessage = (ev) => {
    // this.update(ev);
    console.log('message event', ev);
  };

  handleClose = (ev) => {
    console.log('onclose event', ev);
  };
}
