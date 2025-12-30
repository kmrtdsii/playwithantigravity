import React from 'react';
import ErrorBoundary from './components/common/ErrorBoundary';
import AppLayout from './components/layout/AppLayout';
import './App.css'; // Ensure default styles are loaded

function App() {
  return (
    <ErrorBoundary>
      <React.Suspense fallback={<div className="loading-screen">Loading...</div>}>
        <AppLayout />
      </React.Suspense>
    </ErrorBoundary>
  );
}

export default App;
