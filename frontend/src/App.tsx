import React from 'react';
import { GitProvider } from './context/GitAPIContext';
import GitGraphViz from './components/visualization/GitGraphViz';
import './App.css'; // Ensure default styles are loaded

function App() {
  return (
    <GitProvider>
      <div className="app-container" style={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column' }}>
        <header style={{ padding: '1rem', borderBottom: '1px solid #ccc' }}>
          <h1>GitForge</h1>
        </header>
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <GitGraphViz />
        </div>
      </div>
    </GitProvider>
  );
}

export default App;
