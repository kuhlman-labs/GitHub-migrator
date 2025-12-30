import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, useLocation, useParams } from 'react-router-dom';
import { ThemeProvider, BaseStyles } from '@primer/react';
import { Dashboard } from './components/Dashboard';
import { RepositoryDetail } from './components/RepositoryDetail';
import { Repositories } from './components/Repositories';
import { Analytics } from './components/Analytics';
import { BatchManagement } from './components/BatchManagement';
import { BatchBuilderPage } from './components/BatchManagement/BatchBuilderPage';
import { MigrationHistory } from './components/MigrationHistory';
import { Dependencies } from './components/Dependencies';
import { UserMappingTable } from './components/UserMapping';
import { TeamMappingTable } from './components/TeamMapping';
import { SourcesPage } from './components/Sources';
import { SettingsPage } from './components/Settings';
import { Navigation } from './components/common/Navigation';
import { PageLayout } from './components/common/PageLayout';
import { Login } from './components/Auth/Login';
import { ProtectedRoute } from './components/Auth/ProtectedRoute';
import { Setup } from './components/Setup';
import { ToastProvider } from './contexts/ToastContext';
import { SourceProvider } from './contexts/SourceContext';
import { useSetupStatus } from './hooks/useQueries';

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
                      <SourceProvider>
                        <Navigation />
                        <ProtectedRoutes />
                      </SourceProvider>
                    </SetupCheck>
                  </ProtectedRoute>
                } />
              </Routes>
            </div>
          </ToastProvider>
        </Router>
      </BaseStyles>
    </ThemeProvider>
  );
}

// SetupCheck component redirects to /setup if setup is not complete
function SetupCheck({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const { data: setupStatus, isLoading, isError } = useSetupStatus();

  // Don't redirect if we're already on the setup page
  if (location.pathname === '/setup') {
    return <>{children}</>;
  }

  // Show loading state
  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        Loading...
      </div>
    );
  }

  // Redirect to setup if not complete or if we couldn't fetch status
  if (isError || !setupStatus?.setup_completed) {
    return <Navigate to="/setup" replace />;
  }

  return <>{children}</>;
}

// Redirect component for /org/:orgName to /repositories?organization=:orgName
function OrgRedirect() {
  const { orgName } = useParams<{ orgName: string }>();
  const encodedOrg = encodeURIComponent(orgName || '');
  return <Navigate to={`/repositories?organization=${encodedOrg}`} replace />;
}

// Redirect component for /org/:orgName/project/:projectName
function OrgProjectRedirect() {
  const { orgName, projectName } = useParams<{ orgName: string; projectName: string }>();
  const encodedOrg = encodeURIComponent(orgName || '');
  const encodedProject = encodeURIComponent(projectName || '');
  return <Navigate to={`/repositories?organization=${encodedOrg}&project=${encodedProject}`} replace />;
}

function ProtectedRoutes() {
  return (
    <Routes>
      {/* Full-width pages (no container) */}
      <Route path="/batches/new" element={<PageLayout fullWidth><BatchBuilderPage /></PageLayout>} />
      <Route path="/batches/:batchId/edit" element={<PageLayout fullWidth><BatchBuilderPage /></PageLayout>} />
      
      {/* Standard pages (with max-width container and responsive padding) */}
      <Route path="/" element={<PageLayout><Dashboard /></PageLayout>} />
      
      {/* Redirect old organization detail routes to repositories view with filters */}
      <Route path="/org/:orgName" element={<OrgRedirect />} />
      <Route path="/org/:orgName/project/:projectName" element={<OrgProjectRedirect />} />
      
      <Route path="/repository/:fullName" element={<PageLayout><RepositoryDetail /></PageLayout>} />
      <Route path="/analytics" element={<PageLayout><Analytics /></PageLayout>} />
      <Route path="/repositories" element={<PageLayout><Repositories /></PageLayout>} />
      <Route path="/dependencies" element={<PageLayout><Dependencies /></PageLayout>} />
      <Route path="/batches" element={<PageLayout><BatchManagement /></PageLayout>} />
      <Route path="/history" element={<PageLayout><MigrationHistory /></PageLayout>} />
      <Route path="/user-mappings" element={<PageLayout><UserMappingTable /></PageLayout>} />
      <Route path="/team-mappings" element={<PageLayout><TeamMappingTable /></PageLayout>} />
      <Route path="/sources" element={<PageLayout><SourcesPage /></PageLayout>} />
      <Route path="/settings" element={<PageLayout><SettingsPage /></PageLayout>} />
    </Routes>
  );
}

export default App;

