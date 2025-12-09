import { GitProvider } from './context/GitAPIContext';
import AppLayout from './components/layout/AppLayout';
import './App.css'; // Ensure default styles are loaded

function App() {
  return (
    <GitProvider>
      <AppLayout />
    </GitProvider>
  );
}

export default App;
