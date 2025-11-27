import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom';
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
import { Setup } from './components/Setup';
import { AuthProvider } from './contexts/AuthContext';
import { ToastProvider } from './contexts/ToastContext';
import { api } from './services/api';

const THEME_STORAGE_KEY = 'primer-theme-mode';

function App() {
  // Initialize theme from localStorage or default to 'day'
  const [colorMode] = useState<'day' | 'night'>(() => {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    return (stored === 'day' || stored === 'night') ? stored : 'day';
  });

  // Set Primer data attributes on document root for theming
  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-color-mode', colorMode === 'day' ? 'light' : 'dark');
    root.setAttribute('data-light-theme', 'light');
    root.setAttribute('data-dark-theme', 'dark');
  }, [colorMode]);

  return (
    <ThemeProvider colorMode={colorMode} preventSSRMismatch>
      <BaseStyles>
        <Router>
          <AuthProvider>
            <ToastProvider>
              <div className="min-h-screen" style={{ backgroundColor: 'var(--bgColor-muted)', color: 'var(--fgColor-default)' }}>
                <Routes>
                  {/* Setup page (public) */}
                  <Route path="/setup" element={<Setup />} />
                  
                  {/* Login page (public) */}
                  <Route path="/login" element={<Login />} />
                  
                  {/* Protected routes with navigation and setup check */}
                  <Route path="*" element={
                    <ProtectedRoute>
                      <SetupCheck>
                        <Navigation />
                        <ProtectedRoutes />
                      </SetupCheck>
                    </ProtectedRoute>
                  } />
                </Routes>
              </div>
            </ToastProvider>
          </AuthProvider>
        </Router>
      </BaseStyles>
    </ThemeProvider>
  );
}

// SetupCheck component redirects to /setup if setup is not complete
function SetupCheck({ children }: { children: React.ReactNode }) {
  const [setupComplete, setSetupComplete] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const location = useLocation();

  useEffect(() => {
    // Skip check if we're on the setup page to avoid unnecessary API calls
    if (location.pathname === '/setup') {
      return;
    }

    const checkSetup = async () => {
      setLoading(true);
      try {
        const status = await api.getSetupStatus();
        setSetupComplete(status.setup_completed);
      } catch (error) {
        console.error('Failed to check setup status:', error);
        // If we can't check setup status, assume it's not complete
        setSetupComplete(false);
      } finally {
        setLoading(false);
      }
    };

    checkSetup();
  }, [location.pathname]); // Re-check when route changes

  // Don't redirect if we're already on the setup page
  if (location.pathname === '/setup') {
    return <>{children}</>;
  }

  // Show loading state
  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        Loading...
      </div>
    );
  }

  // Redirect to setup if not complete
  if (!setupComplete) {
    return <Navigate to="/setup" replace />;
  }

  return <>{children}</>;
}

function ProtectedRoutes() {
  return (
    <Routes>
          {/* Full-width pages (no container) */}
          <Route path="/batches/new" element={<BatchBuilderPage />} />
          <Route path="/batches/:batchId/edit" element={<BatchBuilderPage />} />
          
          {/* Standard pages (with max-width container and responsive padding) */}
          <Route path="/" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <Dashboard />
            </main>
          } />
          <Route path="/org/:orgName" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <OrganizationDetail />
            </main>
          } />
          <Route path="/org/:orgName/project/:projectName" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <OrganizationDetail />
            </main>
          } />
          <Route path="/repository/:fullName" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <RepositoryDetail />
            </main>
          } />
          <Route path="/analytics" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <Analytics />
            </main>
          } />
          <Route path="/repositories" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <Repositories />
            </main>
          } />
          <Route path="/batches" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <BatchManagement />
            </main>
          } />
          <Route path="/history" element={
            <main id="main-content" className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <MigrationHistory />
            </main>
          } />
        </Routes>
  );
}

export default App;

