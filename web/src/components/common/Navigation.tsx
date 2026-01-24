import { Link, useLocation, useSearchParams } from 'react-router-dom';
import { TextInput } from '@primer/react';
import { MarkGithubIcon, SearchIcon, GearIcon, CopilotIcon } from '@primer/octicons-react';
import { IconButton } from '@primer/react';
import { UserProfile } from './UserProfile';
import { SourceSelector } from './SourceSelector';

export function Navigation() {
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  
  const isActive = (path: string) => location.pathname === path;
  
  const linkClass = (path: string) =>
    `px-3 py-2 text-sm font-semibold transition-colors rounded-md ${
      isActive(path)
        ? ''
        : 'hover:bg-[var(--control-bgColor-hover)]'
    }`;
  
  // Get context-aware search placeholder and current value
  const getSearchContext = () => {
    const path = location.pathname;
    
    if (path === '/') {
      return {
        placeholder: 'Search organizations...',
        searchParam: 'search',
        isSearchable: true,
      };
    } else if (path === '/repositories') {
      return {
        placeholder: 'Search repositories...',
        searchParam: 'search',
        isSearchable: true,
      };
    } else if (path === '/batches') {
      return {
        placeholder: 'Search batches...',
        searchParam: 'search',
        isSearchable: true,
      };
    } else if (path === '/history') {
      return {
        placeholder: 'Search migration history...',
        searchParam: 'search',
        isSearchable: true,
      };
    } else if (path === '/user-mappings') {
      return {
        placeholder: 'Search user mappings...',
        searchParam: 'search',
        isSearchable: true,
      };
    } else if (path === '/team-mappings') {
      // Team mappings page has its own in-page search
      return {
        placeholder: '',
        searchParam: '',
        isSearchable: false,
      };
    } else if (path === '/dependencies') {
      // Dependencies page has its own in-page search
      return {
        placeholder: '',
        searchParam: '',
        isSearchable: false,
      };
    } else if (path.startsWith('/org/')) {
      // For organization detail pages
      // Note: This could be repositories OR Azure DevOps projects depending on the org type
      // "Search repositories..." works as a reasonable default for both contexts
      return {
        placeholder: 'Search repositories...',
        searchParam: 'search',
        isSearchable: true,
      };
    }
    
    // For pages without search (Analytics, Setup, etc.)
    return {
      placeholder: '',
      searchParam: '',
      isSearchable: false,
    };
  };
  
  const searchContext = getSearchContext();
  const currentSearch = searchParams.get(searchContext.searchParam) || '';
  
  const handleSearchChange = (value: string) => {
    const newParams = new URLSearchParams(searchParams);
    if (value.trim()) {
      newParams.set(searchContext.searchParam, value.trim());
    } else {
      newParams.delete(searchContext.searchParam);
    }
    setSearchParams(newParams);
  };
  
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
              
              {/* Main Navigation Links - Organized by workflow phase */}
              <div className="flex items-center">
                {/* Overview */}
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
                
                {/* Separator: Overview → Explore */}
                <div className="w-px h-4 mx-2" style={{ backgroundColor: 'var(--borderColor-muted)' }} />
                
                {/* Explore Phase */}
                <Link 
                  to="/repositories" 
                  className={linkClass('/repositories')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/repositories') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                  Repositories
                </Link>
                <Link 
                  to="/dependencies" 
                  className={linkClass('/dependencies')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/dependencies') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                  Dependencies
                </Link>
                
                {/* Separator: Explore → Configure */}
                <div className="w-px h-4 mx-2" style={{ backgroundColor: 'var(--borderColor-muted)' }} />
                
                {/* Configure Phase */}
                <Link 
                  to="/user-mappings" 
                  className={linkClass('/user-mappings')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/user-mappings') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                  Users
                </Link>
                <Link 
                  to="/team-mappings" 
                  className={linkClass('/team-mappings')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/team-mappings') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                >
                  Teams
                </Link>
                
                {/* Separator: Configure → Execute */}
                <div className="w-px h-4 mx-2" style={{ backgroundColor: 'var(--borderColor-muted)' }} />
                
                {/* Execute Phase */}
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
                  to="/copilot" 
                  className={linkClass('/copilot')}
                  style={{ 
                    color: 'var(--fgColor-default)',
                    backgroundColor: isActive('/copilot') ? 'var(--bgColor-neutral-muted)' : 'transparent'
                  }}
                  title="AI-powered migration assistant"
                >
                  <CopilotIcon size={16} className="mr-1" />
                  Copilot
                </Link>
                
                {/* Separator: Execute → Report */}
                <div className="w-px h-4 mx-2" style={{ backgroundColor: 'var(--borderColor-muted)' }} />
                
                {/* Report Phase */}
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
          
            {/* Navigation End - Right Side Utilities */}
          <div className="flex items-center gap-4">
            {/* Content Filters (always before utility icons) */}
            <div className="flex items-center gap-3">
              {/* Source Selector - content filter, shown before settings */}
              <SourceSelector />
              
              {/* Context-Aware Global Search - content filter */}
              {searchContext.isSearchable && (
                <TextInput
                  leadingVisual={SearchIcon}
                  placeholder={searchContext.placeholder}
                  value={currentSearch}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  style={{ width: 300 }}
                />
              )}
            </div>
            
            {/* Utility Icons (fixed position on right) */}
            <div className="flex items-center gap-2">
              <Link to="/settings">
                <IconButton 
                  icon={GearIcon} 
                  aria-label="Settings"
                  variant="invisible"
                />
              </Link>
              <UserProfile />
            </div>
          </div>
        </div>
      </div>
    </nav>
    </>
  );
}

