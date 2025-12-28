export interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: React.ReactNode;
}

/**
 * A reusable page header component with consistent styling.
 * Use at the top of pages for title, description, and action buttons.
 *
 * @example
 * <PageHeader
 *   title="Dashboard"
 *   description="Overview of migration progress"
 *   actions={
 *     <Button variant="primary" onClick={handleAction}>
 *       Start Discovery
 *     </Button>
 *   }
 * />
 */
export function PageHeader({ title, description, actions }: PageHeaderProps) {
  return (
    <div className="flex justify-between items-start mb-8">
      <div>
        <h1
          className="text-2xl font-semibold"
          style={{ color: 'var(--fgColor-default)' }}
        >
          {title}
        </h1>
        {description && (
          <p
            className="text-sm mt-1"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            {description}
          </p>
        )}
      </div>
      {actions && <div className="flex items-center gap-4">{actions}</div>}
    </div>
  );
}

