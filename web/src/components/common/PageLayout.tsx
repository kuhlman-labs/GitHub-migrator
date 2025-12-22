import { RefreshIndicator } from './RefreshIndicator';

export interface PageLayoutProps {
  children: React.ReactNode;
  /** Whether the page is currently refreshing/fetching data */
  isRefreshing?: boolean;
  /** Whether to use full width (no max-width constraint) */
  fullWidth?: boolean;
  /** Custom className for the main content wrapper */
  className?: string;
}

/**
 * A reusable page layout component that provides consistent structure.
 * Wraps content with max-width container, responsive padding, and optional refresh indicator.
 *
 * @example
 * <PageLayout isRefreshing={isFetching}>
 *   <PageHeader title="Dashboard" description="Overview of migration progress" />
 *   <YourContent />
 * </PageLayout>
 */
export function PageLayout({
  children,
  isRefreshing = false,
  fullWidth = false,
  className = '',
}: PageLayoutProps) {
  return (
    <main
      id="main-content"
      className={`${fullWidth ? '' : 'max-w-[1920px] mx-auto'} px-4 sm:px-6 lg:px-8 py-8 relative ${className}`}
    >
      <RefreshIndicator isRefreshing={isRefreshing} />
      {children}
    </main>
  );
}

