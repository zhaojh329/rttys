module.exports = {
  publicPath:'/',
  devServer: {
    proxy: {
      '/devs': {
        target: 'http://127.0.0.1:5913'
      },
      '/signin': {
        target: 'http://127.0.0.1:5913'
      },
      '/cmd': {
        target: 'http://127.0.0.1:5913'
      },
      '/ws': {
        ws: true,
        target: 'http://127.0.0.1:5913'
      }
    }
  }
}
