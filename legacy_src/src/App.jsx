import React from 'react';
import AppLayout from './components/layout/AppLayout';

import { GitProvider } from './lib/GitContext';

function App() {
  return (
    <GitProvider>
      <AppLayout />
    </GitProvider>
  );
}

export default App;
