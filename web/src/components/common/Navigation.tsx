import { Link, useLocation } from 'react-router-dom';
import { MarkGithubIcon } from '@primer/octicons-react';
import { UserProfile } from './UserProfile';

export function Navigation() {
  const location = useLocation();
  
  const isActive = (path: string) => location.pathname === path;
  
  const linkClass = (path: string) =>
    `px-4 py-2 font-medium transition-colors border-b-2 ${
      isActive(path)
        ? 'text-white border-white'
        : 'text-white/90 border-transparent hover:text-white hover:border-gray-300'
    } !important`;
  
  return (
    <nav className="bg-gh-header-bg border-b border-white/10">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center space-x-6">
            <Link to="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
              <MarkGithubIcon size={32} className="text-white" />
              <span className="text-gh-header-text font-semibold text-base">
                Migrator
              </span>
            </Link>
            <div className="flex space-x-1 ml-4">
              <Link to="/" className={linkClass('/')} style={{ color: isActive('/') ? '#ffffff' : 'rgba(255, 255, 255, 0.9)' }}>
                Dashboard
              </Link>
              <Link to="/analytics" className={linkClass('/analytics')} style={{ color: isActive('/analytics') ? '#ffffff' : 'rgba(255, 255, 255, 0.9)' }}>
                Analytics
              </Link>
              <Link to="/batches" className={linkClass('/batches')} style={{ color: isActive('/batches') ? '#ffffff' : 'rgba(255, 255, 255, 0.9)' }}>
                Batches
              </Link>
              <Link to="/history" className={linkClass('/history')} style={{ color: isActive('/history') ? '#ffffff' : 'rgba(255, 255, 255, 0.9)' }}>
                History
              </Link>
            </div>
          </div>
          
          {/* User Profile */}
          <div className="flex items-center">
            <UserProfile />
          </div>
        </div>
      </div>
    </nav>
  );
}

