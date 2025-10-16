import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Dashboard } from './components/Dashboard';
import { OrganizationDetail } from './components/OrganizationDetail';
import { RepositoryDetail } from './components/RepositoryDetail';
import { Analytics } from './components/Analytics';
import { BatchManagement } from './components/BatchManagement';
import { BatchBuilderPage } from './components/BatchManagement/BatchBuilderPage';
import { SelfServiceMigration } from './components/SelfService';
import { MigrationHistory } from './components/MigrationHistory';
import { Navigation } from './components/common/Navigation';

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gh-canvas-default">
        <Navigation />
        <Routes>
          {/* Full-width pages (no container) */}
          <Route path="/batches/new" element={<BatchBuilderPage />} />
          <Route path="/batches/:batchId/edit" element={<BatchBuilderPage />} />
          
          {/* Standard pages (with container) */}
          <Route path="/" element={
            <main className="container mx-auto px-4 py-8">
              <Dashboard />
            </main>
          } />
          <Route path="/org/:orgName" element={
            <main className="container mx-auto px-4 py-8">
              <OrganizationDetail />
            </main>
          } />
          <Route path="/repository/:fullName" element={
            <main className="container mx-auto px-4 py-8">
              <RepositoryDetail />
            </main>
          } />
          <Route path="/analytics" element={
            <main className="container mx-auto px-4 py-8">
              <Analytics />
            </main>
          } />
          <Route path="/batches" element={
            <main className="container mx-auto px-4 py-8">
              <BatchManagement />
            </main>
          } />
          <Route path="/self-service" element={
            <main className="container mx-auto px-4 py-8">
              <SelfServiceMigration />
            </main>
          } />
          <Route path="/history" element={
            <main className="container mx-auto px-4 py-8">
              <MigrationHistory />
            </main>
          } />
        </Routes>
      </div>
    </Router>
  );
}

export default App;

