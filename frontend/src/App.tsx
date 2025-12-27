import React from 'react';
import { GitProvider } from './context/GitAPIContext';
import ErrorBoundary from './components/common/ErrorBoundary';
import AppLayout from './components/layout/AppLayout';
import { useTranslation } from 'react-i18next';
import './App.css'; // Ensure default styles are loaded

function App() {
  const { t } = useTranslation();
  return (
    <GitProvider>
      <ErrorBoundary>
        <React.Suspense fallback={<div className="loading-screen">{t('app.loading')}</div>}>
          <AppLayout />
        </React.Suspense>
      </ErrorBoundary>
    </GitProvider>
  );
}

export default App;
