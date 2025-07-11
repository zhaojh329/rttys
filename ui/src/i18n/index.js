/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

import { createI18n } from 'vue-i18n'
import en from './en.json'
import zhCN from './zh-CN.json'

const i18n = createI18n({
  locale: navigator.language === 'zh-CN' ? 'zh-CN' : 'en',
  fallbackLocale: 'en',
  silentTranslationWarn: true,
  silentFallbackWarn: true,
  messages: {
    en,
    'zh-CN': zhCN
  }
})

export default i18n
