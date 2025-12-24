import React from 'react';
import { GitProvider } from './context/GitAPIContext';
import ErrorBoundary from './components/common/ErrorBoundary';
import AppLayout from './components/layout/AppLayout';
import './App.css'; // Ensure default styles are loaded

function App() {
  return (
    <GitProvider>
      <ErrorBoundary>
        <React.Suspense fallback={<div className="loading-screen">Loading GitGym...</div>}>
          <AppLayout />
        </React.Suspense>
      </ErrorBoundary>
    </GitProvider>
  );
}

export default App;
