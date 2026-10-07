export type ThemeChoice = 'system' | 'light' | 'dark'
export const themeKey = 'unicourse:theme'

/** Stored choice; anything else (absent, garbage, storage blocked) is 'system'. */
export function read(): ThemeChoice {
  try {
    const v = localStorage.getItem(themeKey)
    return v === 'light' || v === 'dark' ? v : 'system'
  } catch {
    return 'system'
  }
}

/** 'system' removes the key. Storage failures are ignored: the choice then lasts for this page only. */
export function write(choice: ThemeChoice): void {
  try {
    if (choice === 'system') localStorage.removeItem(themeKey)
    else localStorage.setItem(themeKey, choice)
  } catch {
    // storage blocked: the choice applies to this page only
  }
}

/** 'system' removes data-theme (CSS follows prefers-color-scheme); otherwise sets it. */
export function apply(choice: ThemeChoice, root: HTMLElement = document.documentElement): void {
  if (choice === 'system') root.removeAttribute('data-theme')
  else root.setAttribute('data-theme', choice)
}

/** apply(read()). public/theme-boot.js is this function, hand-minified; keep them in step. */
export function boot(): void {
  apply(read())
}
