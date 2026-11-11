import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from '@/App'
import { t } from '@/i18n'
import '@/index.css'

document.title = t.app.title

const ROOT_ELEMENT_ID = 'root'

const container = document.getElementById(ROOT_ELEMENT_ID)
// index.html always contains the root element; without it there is nothing to render into.
if (container !== null) {
  createRoot(container).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}
