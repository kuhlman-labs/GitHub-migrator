import { Button, IconButton } from '@primer/react';
import type { ButtonProps } from '@primer/react';
import { XIcon } from '@primer/octicons-react';
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
 * PrimaryButton - Blue button for primary actions
 * Use for: "Start Discovery", "Submit", "Create", main call-to-action buttons
 */
export const PrimaryButton = forwardRef<HTMLButtonElement, SharedButtonProps>(
  function PrimaryButton({ children, ...props }, ref) {
    return (
      <Button
        ref={ref}
        variant="primary"
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
interface CloseIconButtonProps {
  'aria-label'?: string;
  onClick?: () => void;
  disabled?: boolean;
  size?: 'small' | 'medium' | 'large';
  className?: string;
}

export const CloseIconButton = forwardRef<HTMLButtonElement, CloseIconButtonProps>(
  function CloseIconButton({ 'aria-label': ariaLabel = 'Close', onClick, disabled, size, className }, ref) {
    return (
      <IconButton
        ref={ref}
        icon={XIcon}
        variant="invisible"
        aria-label={ariaLabel}
        onClick={onClick}
        disabled={disabled}
        size={size}
        className={className}
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

