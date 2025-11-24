import { Link, useLocation } from 'react-router-dom';
import { MarkGithubIcon } from '@primer/octicons-react';
import { UserProfile } from './UserProfile';

export function Navigation() {
  const location = useLocation();
  
  const isActive = (path: string) => location.pathname === path;
  
  const linkClass = (path: string) =>
    `px-3 py-2 text-sm font-semibold transition-colors rounded-md ${
      isActive(path)
        ? ''
        : 'hover:bg-[var(--control-bgColor-hover)]'
    }`;
  
  return (
    <>
      {/* Skip link for accessibility */}
      <a 
        href="#main-content" 
        className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-4 focus:py-2 focus:rounded"
        style={{ 
          backgroundColor: 'var(--bgColor-accent-emphasis)',
          color: 'var(--fgColor-onEmphasis)'
        }}
      >
        Skip to main content
      </a>
      
      <nav 
        className="border-b"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderColor: 'var(--borderColor-muted)',
          color: 'var(--fgColor-default)'
        }}
        aria-label="Main navigation"
      >
        <div className="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16 gap-4">
            {/* Navigation Start */}
            <div className="flex items-center gap-4">
              <Link 
                to="/" 
                className="flex items-center gap-2 hover:opacity-80 transition-opacity"
                style={{ color: 'var(--fgColor-default)' }}
              >
                <MarkGithubIcon size={32} />
                <span className="font-semibold text-base">
                Migrator
              </span>
            </Link>
              
              {/* Main Navigation Links */}
              <div className="flex items-center gap-2">
                <Link 
                  to="/" 
                  className={linkClass('/')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                Dashboard
              </Link>
                <Link 
                  to="/analytics" 
                  className={linkClass('/analytics')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/analytics') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                Analytics
              </Link>
                <Link 
                  to="/batches" 
                  className={linkClass('/batches')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/batches') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                Batches
              </Link>
                <Link 
                  to="/history" 
                  className={linkClass('/history')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/history') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                History
              </Link>
            </div>
          </div>
          
            {/* Navigation End */}
          <div className="flex items-center">
            <UserProfile />
          </div>
        </div>
      </div>
    </nav>
    </>
  );
}

