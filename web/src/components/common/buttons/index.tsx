import { Button, IconButton } from '@primer/react';
import type { ButtonProps, IconButtonProps } from '@primer/react';
import { forwardRef } from 'react';

/**
 * Shared button components that extend Primer Button with additional variants.
 * These components use CSS classes from index.css to override Primer styles.
 */

type SharedButtonProps = Omit<ButtonProps, 'variant'> & {
  variant?: ButtonProps['variant'];
};

/**
 * SuccessButton - Green button for positive/confirmation actions
 * Use for: "Start Migration", "Create & Start", "Confirm", "Save"
 */
export const SuccessButton = forwardRef<HTMLButtonElement, SharedButtonProps>(
  function SuccessButton({ children, className, ...props }, ref) {
    return (
      <Button
        ref={ref}
        className={`btn-success ${className || ''}`}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

/**
 * AttentionButton - Yellow/orange button for warning actions
 * Use for: "Mark as Won't Migrate", "Skip", "Pause"
 */
export const AttentionButton = forwardRef<HTMLButtonElement, SharedButtonProps>(
  function AttentionButton({ children, className, ...props }, ref) {
    return (
      <Button
        ref={ref}
        className={`btn-attention ${className || ''}`}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

/**
 * BorderedButton - Invisible variant with a visible border
 * Use for: Secondary actions, filter buttons, toolbar actions
 */
export const BorderedButton = forwardRef<HTMLButtonElement, SharedButtonProps>(
  function BorderedButton({ children, className, ...props }, ref) {
    return (
      <Button
        ref={ref}
        variant="invisible"
        className={`btn-bordered ${className || ''}`}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

/**
 * SecondaryButton - Default variant button for secondary actions
 * Use for: "Cancel", "Close", "Back"
 */
export const SecondaryButton = forwardRef<HTMLButtonElement, SharedButtonProps>(
  function SecondaryButton({ children, className, ...props }, ref) {
    return (
      <Button
        ref={ref}
        className={`btn-secondary ${className || ''}`}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

/**
 * CloseIconButton - Standardized close button for dialogs and panels
 * Use for: Dialog close buttons, panel close buttons
 */
export const CloseIconButton = forwardRef<HTMLButtonElement, Omit<IconButtonProps, 'icon' | 'aria-label'> & { 'aria-label'?: string }>(
  function CloseIconButton({ 'aria-label': ariaLabel = 'Close', ...props }, ref) {
    return (
      <IconButton
        ref={ref}
        variant="invisible"
        aria-label={ariaLabel}
        {...props}
      />
    );
  }
);

/**
 * FilterDropdownButton - Full-width button for filter dropdowns
 * Use for: Filter dropdowns in FilterBar
 */
export const FilterDropdownButton = forwardRef<HTMLButtonElement, SharedButtonProps>(
  function FilterDropdownButton({ children, className, ...props }, ref) {
    return (
      <Button
        ref={ref}
        variant="invisible"
        className={`btn-filter-dropdown ${className || ''}`}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

// Re-export Primer Button and IconButton for convenience
export { Button, IconButton } from '@primer/react';

