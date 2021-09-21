const CompressionPlugin = require('compression-webpack-plugin')

module.exports = {
  productionSourceMap: false,
  chainWebpack: config => {
    config.plugin('html')
      .tap(args => {
        args[0].title = 'Rttys';
        return args;
      })
  },
  configureWebpack: config => {
    if (process.env.NODE_ENV === 'production') {
      return {
        plugins: [new CompressionPlugin({
          test: /\.js$|\.css$/,
          threshold: 4096,
          deleteOriginalAssets: true,
          filename: '[path][base]?gz'
        })]
      }
    }
  },
  pluginOptions: {
    i18n: {
      locale: 'en',
      fallbackLocale: 'en',
      localeDir: 'locales',
      enableInSFC: false
    }
  },
  devServer: {
    proxy: {
      '/devs': {
        target: 'http://127.0.0.1:5913'
      },
      '/signin': {
        target: 'http://127.0.0.1:5913'
      },
      '/signout': {
        target: 'http://127.0.0.1:5913'
      },
      '/alive': {
        target: 'http://127.0.0.1:5913'
      },
      '/signup': {
        target: 'http://127.0.0.1:5913'
      },
      '/isadmin': {
        target: 'http://127.0.0.1:5913'
      },
      '/users': {
        target: 'http://127.0.0.1:5913'
      },
      '/bind': {
        target: 'http://127.0.0.1:5913'
      },
      '/unbind': {
        target: 'http://127.0.0.1:5913'
      },
      '/delete': {
        target: 'http://127.0.0.1:5913'
      },
      '/cmd/*': {
        target: 'http://127.0.0.1:5913'
      },
      '/connect/*': {
        ws: true,
        target: 'http://127.0.0.1:5913'
      },
      '/fontsize/*': {
        target: 'http://127.0.0.1:5913'
      },
      '/authorized/*': {
        target: 'http://127.0.0.1:5913'
      },
      '/web/*': {
        target: 'http://127.0.0.1:5913'
      },
      '/file/*': {
        target: 'http://127.0.0.1:5913'
      }
    }
  }
}