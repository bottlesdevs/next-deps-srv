import { defineStore } from 'pinia'
import { ref, computed, watchEffect } from 'vue'

const STORAGE_KEY = 'theme'
const media = window.matchMedia('(prefers-color-scheme: dark)')

export const MODES = ['system', 'light', 'dark']

// 'system' follows the OS; 'light'/'dark' pin it.
export const useThemeStore = defineStore('theme', () => {
  const stored = localStorage.getItem(STORAGE_KEY)
  const mode = ref(MODES.includes(stored) ? stored : 'system')

  // Mirror the media query into a ref. Reading media.matches directly inside
  // a computed would not be reactive, so the OS switching themes would never
  // invalidate it and the page would keep its stale theme.
  const systemDark = ref(media.matches)
  media.addEventListener('change', e => { systemDark.value = e.matches })

  const resolved = computed(() =>
    mode.value === 'system' ? (systemDark.value ? 'dark' : 'light') : mode.value
  )
  const isDark = computed(() => resolved.value === 'dark')

  // Drives both our own tokens (data-theme) and PrimeVue's darkModeSelector
  // ('.dark'), so custom CSS and PrimeVue components stay in step. Runs
  // whenever mode or the OS preference changes.
  watchEffect(() => {
    document.documentElement.dataset.theme = resolved.value
    document.documentElement.classList.toggle('dark', isDark.value)
  })

  function setMode(next) {
    if (!MODES.includes(next)) return
    mode.value = next
    if (next === 'system') localStorage.removeItem(STORAGE_KEY)
    else localStorage.setItem(STORAGE_KEY, next)
  }

  // Cycle system -> light -> dark -> system, so returning to "follow the OS"
  // is always reachable from the UI.
  function toggle() {
    setMode(MODES[(MODES.indexOf(mode.value) + 1) % MODES.length])
  }

  const icon = computed(() => ({
    system: 'pi pi-desktop',
    light: 'pi pi-sun',
    dark: 'pi pi-moon',
  }[mode.value]))

  const label = computed(() => ({
    system: `Theme: follows system (${resolved.value})`,
    light: 'Theme: light',
    dark: 'Theme: dark',
  }[mode.value]))

  return { mode, resolved, isDark, systemDark, icon, label, setMode, toggle }
})
