import { StrictMode, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { GitProvider } from './context/GitAPIContext.tsx'
import { ThemeProvider } from './context/ThemeContext.tsx'
import './i18n';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <GitProvider>
        <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', color: '#888' }}>Loading...</div>}>
          <App />
        </Suspense>
      </GitProvider>
    </ThemeProvider>
  </StrictMode>,
)
