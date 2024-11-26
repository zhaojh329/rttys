/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

import { createI18n } from 'vue-i18n'
import en from './en.json'
import zh from './zh-CN.json'

const i18n = createI18n({
  locale: navigator.language,
  fallbackLocale: 'en-US',
  silentTranslationWarn: true,
  silentFallbackWarn: true,
  messages: {
    'en-US': en,
    'zh-CN': zh
  }
})

export default i18n
