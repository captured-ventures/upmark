// Fonts — bundled, no network dependency.
import '@fontsource-variable/newsreader/standard.css'
import '@fontsource-variable/newsreader/standard-italic.css'
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/500.css'
import '@fontsource/ibm-plex-sans/600.css'
import '@fontsource/ibm-plex-sans/700.css'
import '@fontsource/ibm-plex-sans/400-italic.css'
import '@fontsource/ibm-plex-mono/400.css'
import '@fontsource/ibm-plex-mono/500.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import '@fontsource/jetbrains-mono/400-italic.css'

// Theme-specific typefaces
import '@fontsource/eb-garamond/400.css'
import '@fontsource/eb-garamond/400-italic.css'
import '@fontsource/eb-garamond/500.css'
import '@fontsource-variable/lora/wght.css'
import '@fontsource-variable/lora/wght-italic.css'
import '@fontsource/caveat/400.css'
import '@fontsource/caveat/500.css'
import '@fontsource/caveat/600.css'
import '@fontsource/courier-prime/400.css'
import '@fontsource/courier-prime/400-italic.css'
import '@fontsource/courier-prime/700.css'
import '@fontsource/vt323/400.css'

import './styles/app.css'
import './styles/themes.css'
import './styles/markdown.css'
import 'katex/dist/katex.min.css'
import App from './App.svelte'

const app = new App({
  target: document.getElementById('app')!,
})

export default app
