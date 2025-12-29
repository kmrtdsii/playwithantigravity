import { Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { GitProvider } from './context/GitAPIContext.tsx'
import { MissionProvider } from './context/MissionContext.tsx'
import { ThemeProvider } from './context/ThemeContext.tsx'
import './i18n';

// StrictMode disabled temporarily for debugging timer issue
createRoot(document.getElementById('root')!).render(
  <ThemeProvider>
    <GitProvider>
      <MissionProvider>
        <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', color: '#888' }}>Loading...</div>}>
          <App />
        </Suspense>
      </MissionProvider>
    </GitProvider>
  </ThemeProvider>,
)
