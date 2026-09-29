import { ref } from 'vue'

export type ThemePreference = 'system' | 'light' | 'dark'

const storageKey = 'glosc-theme'
const stored = (() => {
  try { return window.localStorage.getItem(storageKey) }
  catch { return null }
})()
const preference = ref<ThemePreference>(stored === 'light' || stored === 'dark' ? stored : 'system')
const systemDark = window.matchMedia('(prefers-color-scheme: dark)')

function applyTheme() {
  const dark = preference.value === 'dark' || (preference.value === 'system' && systemDark.matches)
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
}

systemDark.addEventListener('change', applyTheme)
applyTheme()

export function useTheme() {
  function setPreference(next: ThemePreference) {
    preference.value = next
    try { window.localStorage.setItem(storageKey, next) }
    catch { /* Theme still works for this visit. */ }
    applyTheme()
  }

  return { preference, setPreference }
}
