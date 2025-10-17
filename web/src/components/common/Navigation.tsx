import { Link, useLocation } from 'react-router-dom';

export function Navigation() {
  const location = useLocation();
  
  const isActive = (path: string) => location.pathname === path;
  
  const linkClass = (path: string) =>
    `px-3 py-2 text-gh-header-text font-medium transition-colors border-b-2 ${
      isActive(path)
        ? 'border-white/50'
        : 'border-transparent hover:text-white/80'
    }`;
  
  return (
    <nav className="bg-gh-header-bg border-b border-white/10">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center space-x-6">
            <Link to="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
              <img 
                src="/github-mark-white.png" 
                alt="GitHub" 
                className="w-8 h-8"
              />
              <span className="text-gh-header-text font-semibold text-base">
                Migrator
              </span>
            </Link>
            <div className="flex space-x-1 ml-4">
              <Link to="/" className={linkClass('/')}>
                Dashboard
              </Link>
              <Link to="/analytics" className={linkClass('/analytics')}>
                Analytics
              </Link>
              <Link to="/batches" className={linkClass('/batches')}>
                Batches
              </Link>
              <Link to="/history" className={linkClass('/history')}>
                History
              </Link>
              <Link to="/self-service" className={linkClass('/self-service')}>
                Self-Service
              </Link>
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
}

