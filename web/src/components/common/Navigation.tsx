import { Link, useLocation } from 'react-router-dom';

export function Navigation() {
  const location = useLocation();
  
  const isActive = (path: string) => location.pathname === path;
  
  const linkClass = (path: string) =>
    `px-4 py-2 rounded-lg transition-colors ${
      isActive(path)
        ? 'bg-blue-600 text-white'
        : 'text-gray-700 hover:bg-gray-100'
    }`;
  
  return (
    <nav className="bg-white shadow-sm border-b border-gray-200">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center space-x-8">
            <h1 className="text-xl font-semibold text-gray-900">
              GitHub Migrator
            </h1>
            <div className="flex space-x-2">
              <Link to="/" className={linkClass('/')}>
                Dashboard
              </Link>
              <Link to="/analytics" className={linkClass('/analytics')}>
                Analytics
              </Link>
              <Link to="/batches" className={linkClass('/batches')}>
                Batches
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

