import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import zhCN from 'element-plus/dist/locale/zh-cn'
import en from 'element-plus/dist/locale/en'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'

export const locale = computed(() => {
  const { locale } = useI18n()
  return locale.value === 'zh-CN' ? zhCN : en
})

export default {
  install(app) {
    app.use(ElementPlus)

    for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
      app.component(key, component)
    }
  }
}
