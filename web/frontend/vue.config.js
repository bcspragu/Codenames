module.exports = {
  devServer: {
    host: '0.0.0.0',
    port: 8081,
    proxy: {
      '/api/': {
        target: 'http://172.17.0.1:8080',
      },
    }
  }
}


