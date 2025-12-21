import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { GitProvider } from './context/GitAPIContext.tsx'
import { ThemeProvider } from './context/ThemeContext.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <GitProvider>
        <App />
      </GitProvider>
    </ThemeProvider>
  </StrictMode>,
)
