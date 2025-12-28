/**
 * Theme constants for consistent styling across components.
 * Maps semantic color names to Primer CSS variables.
 * 
 * Usage:
 *   import { colors, spacing, shadows } from '../styles/theme';
 *   <div style={{ backgroundColor: colors.bgDefault }}>
 */

// Background colors
export const colors = {
  // Primary backgrounds
  bgDefault: 'var(--bgColor-default)',
  bgMuted: 'var(--bgColor-muted)',
  bgInset: 'var(--bgColor-inset)',
  bgEmphasis: 'var(--bgColor-emphasis)',
  
  // Accent backgrounds
  bgAccentEmphasis: 'var(--bgColor-accent-emphasis)',
  bgAccentMuted: 'var(--bgColor-accent-muted)',
  bgNeutralMuted: 'var(--bgColor-neutral-muted)',
  
  // Status backgrounds
  bgSuccess: 'var(--bgColor-success-muted)',
  bgSuccessEmphasis: 'var(--bgColor-success-emphasis)',
  bgDanger: 'var(--bgColor-danger-muted)',
  bgDangerEmphasis: 'var(--bgColor-danger-emphasis)',
  bgWarning: 'var(--bgColor-attention-muted)',
  bgWarningEmphasis: 'var(--bgColor-attention-emphasis)',
  
  // Foreground colors
  fgDefault: 'var(--fgColor-default)',
  fgMuted: 'var(--fgColor-muted)',
  fgOnEmphasis: 'var(--fgColor-onEmphasis)',
  fgAccent: 'var(--fgColor-accent)',
  fgSuccess: 'var(--fgColor-success)',
  fgDanger: 'var(--fgColor-danger)',
  fgWarning: 'var(--fgColor-attention)',
  
  // Border colors
  borderDefault: 'var(--borderColor-default)',
  borderMuted: 'var(--borderColor-muted)',
  borderAccent: 'var(--borderColor-accent-emphasis)',
  borderSuccess: 'var(--borderColor-success-emphasis)',
  borderDanger: 'var(--borderColor-danger-emphasis)',
  
  // Control colors (buttons, inputs)
  controlBgRest: 'var(--control-bgColor-rest)',
  controlBgHover: 'var(--control-bgColor-hover)',
  controlBgActive: 'var(--control-bgColor-active)',
  controlBgDisabled: 'var(--control-bgColor-disabled)',
  
  // Accent emphasis (for primary actions)
  accentEmphasis: 'var(--accent-emphasis)',
  accentSubtle: 'var(--accent-subtle)',
} as const;

// Shadows
export const shadows = {
  small: 'var(--shadow-resting-small)',
  medium: 'var(--shadow-resting-medium)',
  large: 'var(--shadow-floating-large)',
  floating: 'var(--shadow-floating-small)',
} as const;

// Common style objects for reuse
export const cardStyles = {
  base: {
    backgroundColor: colors.bgDefault,
    borderColor: colors.borderDefault,
    borderWidth: '1px',
    borderStyle: 'solid',
    borderRadius: '0.5rem',
    boxShadow: shadows.small,
  },
  selected: {
    borderColor: colors.accentEmphasis,
    backgroundColor: colors.accentSubtle,
  },
} as const;

export const dialogStyles = {
  backdrop: {
    position: 'fixed' as const,
    inset: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    zIndex: 50,
  },
  container: {
    position: 'fixed' as const,
    inset: 0,
    zIndex: 50,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '1rem',
  },
  content: {
    backgroundColor: colors.bgDefault,
    borderRadius: '0.5rem',
    boxShadow: shadows.large,
    maxWidth: '32rem',
    width: '100%',
    maxHeight: '90vh',
    overflow: 'auto',
  },
  header: {
    padding: '1rem',
    borderBottom: `1px solid ${colors.borderDefault}`,
  },
  body: {
    padding: '1rem',
  },
  footer: {
    padding: '0.75rem 1rem',
    borderTop: `1px solid ${colors.borderDefault}`,
    display: 'flex',
    justifyContent: 'flex-end',
    gap: '0.5rem',
  },
} as const;

// Button color presets for custom buttons (prefer Primer Button when possible)
export const buttonColors = {
  success: {
    bg: 'var(--bgColor-success-emphasis)',
    bgHover: 'var(--bgColor-success-emphasis)',
    text: 'var(--fgColor-onEmphasis)',
  },
  danger: {
    bg: 'var(--bgColor-danger-emphasis)',
    bgHover: 'var(--bgColor-danger-emphasis)',
    text: 'var(--fgColor-onEmphasis)',
  },
  primary: {
    bg: 'var(--bgColor-accent-emphasis)',
    bgHover: 'var(--bgColor-accent-emphasis)',
    text: 'var(--fgColor-onEmphasis)',
  },
} as const;

// Badge/Label color mappings
export const badgeColors = {
  custom: {
    bg: colors.fgAccent,
    text: colors.fgOnEmphasis,
    border: colors.fgAccent,
  },
  batchDefault: {
    bg: colors.fgWarning,
    text: colors.fgOnEmphasis,
    border: colors.fgWarning,
  },
  default: {
    bg: colors.fgMuted,
    text: colors.fgOnEmphasis,
    border: colors.fgMuted,
  },
} as const;

