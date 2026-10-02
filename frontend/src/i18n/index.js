import { ref } from 'vue'
import { en } from './en.js'
import { ru } from './ru.js'

const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('ui_locale') : ''
const locale = ref(stored === 'en' || stored === 'ru' ? stored : 'ru')
const dictionaries = { en, ru }

export function t(key, params) {
  const dict = dictionaries[locale.value] || dictionaries.ru
  let str = dict[key] || en[key] || key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      str = str.replace(new RegExp(`\\{${k}\\}`, 'g'), () => v)
    }
  }
  return str
}

export function setLocale(l) {
  if (!dictionaries[l]) return
  locale.value = l
  if (typeof localStorage !== 'undefined') localStorage.setItem('ui_locale', l)
}

export { locale, en, ru }
