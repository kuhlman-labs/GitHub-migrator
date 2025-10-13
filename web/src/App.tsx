import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Dashboard } from './components/Dashboard';
import { OrganizationDetail } from './components/OrganizationDetail';
import { RepositoryDetail } from './components/RepositoryDetail';
import { Analytics } from './components/Analytics';
import { BatchManagement } from './components/BatchManagement';
import { SelfServiceMigration } from './components/SelfService';
import { MigrationHistory } from './components/MigrationHistory';
import { Navigation } from './components/common/Navigation';

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-50">
        <Navigation />
        <main className="container mx-auto px-4 py-8">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/org/:orgName" element={<OrganizationDetail />} />
            <Route path="/repository/:fullName" element={<RepositoryDetail />} />
            <Route path="/analytics" element={<Analytics />} />
            <Route path="/batches" element={<BatchManagement />} />
            <Route path="/self-service" element={<SelfServiceMigration />} />
            <Route path="/history" element={<MigrationHistory />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;

