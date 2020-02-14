import Vue from 'vue'
import Element from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import enLocale from 'element-ui/lib/locale/lang/en'
import zhLocale from 'element-ui/lib/locale/lang/zh-CN'
import i18n from '@/i18n'

i18n.mergeLocaleMessage('en', enLocale);
i18n.mergeLocaleMessage('zh-CN', zhLocale);

Vue.use(Element, {
  i18n: (key, value) => i18n.t(key, value)
});
