import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import zhCN from 'element-plus/dist/locale/zh-cn'
import en from 'element-plus/dist/locale/en'
import i18n from '../i18n'

export const locale = i18n.locale === 'zh-CN' ? zhCN : en

export default {
  install(app) {
    app.use(ElementPlus)

    for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
      app.component(key, component)
    }
  }
}
