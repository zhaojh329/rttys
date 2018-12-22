module.exports = {
  devServer: {
    proxy: {
      '/devs': {
        target: 'http://127.0.0.1:5912'
      },
      '/login': {
        target: 'http://127.0.0.1:5912'
      },
      '/cmd': {
        target: 'http://127.0.0.1:5912'
      }
    }
  }
}
