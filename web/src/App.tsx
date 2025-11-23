import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { ThemeProvider, BaseStyles } from '@primer/react';
import { Dashboard } from './components/Dashboard';
import { OrganizationDetail } from './components/OrganizationDetail';
import { RepositoryDetail } from './components/RepositoryDetail';
import { Repositories } from './components/Repositories';
import { Analytics } from './components/Analytics';
import { BatchManagement } from './components/BatchManagement';
import { BatchBuilderPage } from './components/BatchManagement/BatchBuilderPage';
import { MigrationHistory } from './components/MigrationHistory';
import { Navigation } from './components/common/Navigation';
import { Login } from './components/Auth/Login';
import { ProtectedRoute } from './components/Auth/ProtectedRoute';
import { AuthProvider } from './contexts/AuthContext';

function App() {
  return (
    <ThemeProvider colorMode="light">
      <BaseStyles>
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
      </BaseStyles>
    </ThemeProvider>
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
          <Route path="/org/:orgName/project/:projectName" element={
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
          <Route path="/history" element={
            <main className="container mx-auto px-4 py-8">
              <MigrationHistory />
            </main>
          } />
        </Routes>
  );
}

export default App;

