module.exports = {
  async rewrites() {
    return [
      {
        source: '/api/:slug*',
        destination: 'http://localhost:8080/api/:slug*',
      },
    ];
  },
};
