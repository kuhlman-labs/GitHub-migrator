import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Dashboard } from './components/Dashboard';
import { OrganizationDetail } from './components/OrganizationDetail';
import { RepositoryDetail } from './components/RepositoryDetail';
import { Repositories } from './components/Repositories';
import { Analytics } from './components/Analytics';
import { BatchManagement } from './components/BatchManagement';
import { BatchBuilderPage } from './components/BatchManagement/BatchBuilderPage';
import { SelfServiceMigration } from './components/SelfService';
import { MigrationHistory } from './components/MigrationHistory';
import { Navigation } from './components/common/Navigation';
import { Login } from './components/Auth/Login';
import { ProtectedRoute } from './components/Auth/ProtectedRoute';
import { AuthProvider } from './contexts/AuthContext';

function App() {
  return (
    <Router>
      <AuthProvider>
        <div className="min-h-screen bg-gh-canvas-default">
          <Routes>
            {/* Login page (public) */}
            <Route path="/login" element={<Login />} />
            
            {/* Protected routes with navigation */}
            <Route path="*" element={
              <ProtectedRoute>
                <Navigation />
                <ProtectedRoutes />
              </ProtectedRoute>
            } />
          </Routes>
        </div>
      </AuthProvider>
    </Router>
  );
}

function ProtectedRoutes() {
  return (
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
          <Route path="/repositories" element={
            <main className="container mx-auto px-4 py-8">
              <Repositories />
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
  );
}

export default App;

