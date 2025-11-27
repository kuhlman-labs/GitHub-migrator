/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Override default Tailwind colors with Primer CSS variables
        white: 'var(--bgColor-default)',
        black: 'var(--fgColor-default)',
        gray: {
          50: 'var(--bgColor-muted)',
          100: 'var(--bgColor-muted)',
          200: 'var(--borderColor-muted)',
          300: 'var(--borderColor-default)',
          400: 'var(--borderColor-emphasis)',
          500: 'var(--fgColor-muted)',
          600: 'var(--fgColor-muted)',
          700: 'var(--fgColor-default)',
          800: 'var(--fgColor-default)',
          900: 'var(--fgColor-default)',
        },
        blue: {
          50: 'var(--bgColor-accent-muted)',
          100: 'var(--bgColor-accent-muted)',
          200: 'var(--bgColor-accent-muted)',
          300: 'var(--borderColor-accent-muted)',
          400: 'var(--borderColor-accent-emphasis)',
          500: 'var(--fgColor-accent)',
          600: 'var(--fgColor-accent)',
          700: 'var(--fgColor-accent)',
          800: 'var(--fgColor-accent)',
          900: 'var(--fgColor-accent)',
        },
        green: {
          50: 'var(--bgColor-success-muted)',
          100: 'var(--bgColor-success-muted)',
          200: 'var(--bgColor-success-muted)',
          300: 'var(--borderColor-success-muted)',
          400: 'var(--borderColor-success-emphasis)',
          500: 'var(--fgColor-success)',
          600: 'var(--fgColor-success)',
          700: 'var(--fgColor-success)',
          800: 'var(--bgColor-success-emphasis)',
          900: 'var(--bgColor-success-emphasis)',
        },
        red: {
          50: 'var(--bgColor-danger-muted)',
          100: 'var(--bgColor-danger-muted)',
          200: 'var(--bgColor-danger-muted)',
          300: 'var(--borderColor-danger-muted)',
          400: 'var(--borderColor-danger-emphasis)',
          500: 'var(--fgColor-danger)',
          600: 'var(--fgColor-danger)',
          700: 'var(--fgColor-danger)',
          800: 'var(--bgColor-danger-emphasis)',
          900: 'var(--bgColor-danger-emphasis)',
        },
        yellow: {
          50: 'var(--bgColor-attention-muted)',
          100: 'var(--bgColor-attention-muted)',
          200: 'var(--bgColor-attention-muted)',
          300: 'var(--borderColor-attention-muted)',
          400: 'var(--borderColor-attention-emphasis)',
          500: 'var(--fgColor-attention)',
          600: 'var(--fgColor-attention)',
          700: 'var(--fgColor-attention)',
          800: 'var(--bgColor-attention-emphasis)',
          900: 'var(--bgColor-attention-emphasis)',
        },
        orange: {
          50: 'var(--bgColor-attention-muted)',
          100: 'var(--bgColor-attention-muted)',
          200: 'var(--bgColor-attention-muted)',
          300: 'var(--borderColor-attention-muted)',
          400: 'var(--borderColor-attention-emphasis)',
          500: 'var(--fgColor-attention)',
          600: 'var(--fgColor-attention)',
          700: 'var(--fgColor-attention)',
          800: 'var(--bgColor-attention-emphasis)',
          900: 'var(--bgColor-attention-emphasis)',
        },
        purple: {
          50: 'var(--bgColor-done-muted)',
          100: 'var(--bgColor-done-muted)',
          200: 'var(--bgColor-done-muted)',
          300: 'var(--borderColor-done-muted)',
          400: 'var(--borderColor-done-emphasis)',
          500: 'var(--fgColor-done)',
          600: 'var(--fgColor-done)',
          700: 'var(--fgColor-done)',
          800: 'var(--bgColor-done-emphasis)',
          900: 'var(--bgColor-done-emphasis)',
        },
        pink: {
          50: 'var(--bgColor-sponsors-muted)',
          100: 'var(--bgColor-sponsors-muted)',
          200: 'var(--bgColor-sponsors-muted)',
          300: 'var(--borderColor-sponsors-muted)',
          400: 'var(--borderColor-sponsors-emphasis)',
          500: 'var(--fgColor-sponsors)',
          600: 'var(--fgColor-sponsors)',
          700: 'var(--fgColor-sponsors)',
          800: 'var(--bgColor-sponsors-emphasis)',
          900: 'var(--bgColor-sponsors-emphasis)',
        },
        indigo: {
          50: 'var(--bgColor-accent-muted)',
          100: 'var(--bgColor-accent-muted)',
          200: 'var(--bgColor-accent-muted)',
          300: 'var(--borderColor-accent-muted)',
          400: 'var(--borderColor-accent-emphasis)',
          500: 'var(--fgColor-accent)',
          600: 'var(--fgColor-accent)',
          700: 'var(--fgColor-accent)',
          800: 'var(--bgColor-accent-emphasis)',
          900: 'var(--bgColor-accent-emphasis)',
        },
        teal: {
          50: 'var(--bgColor-success-muted)',
          100: 'var(--bgColor-success-muted)',
          200: 'var(--bgColor-success-muted)',
          300: 'var(--borderColor-success-muted)',
          400: 'var(--borderColor-success-emphasis)',
          500: 'var(--fgColor-success)',
          600: 'var(--fgColor-success)',
          700: 'var(--fgColor-success)',
          800: 'var(--bgColor-success-emphasis)',
          900: 'var(--bgColor-success-emphasis)',
        },
        // Primer CSS Variables - Dynamically adapt to theme
        gh: {
          // Text/Foreground Colors
          'text-primary': 'var(--fgColor-default)',
          'text-secondary': 'var(--fgColor-muted)',
          'text-muted': 'var(--fgColor-muted)',
          'text-link': 'var(--fgColor-accent)',
          'text-on-emphasis': 'var(--fgColor-onEmphasis)',
          
          // Canvas/Background Colors
          'canvas-default': 'var(--bgColor-default)',
          'canvas-raised': 'var(--bgColor-default)',
          'canvas-inset': 'var(--bgColor-inset)',
          'canvas-subtle': 'var(--bgColor-muted)',
          
          // Border Colors
          'border-default': 'var(--borderColor-default)',
          'border-muted': 'var(--borderColor-muted)',
          'border-hover': 'var(--borderColor-emphasis)',
          
          // Header
          'header-bg': 'var(--bgColor-default)',
          'header-text': 'var(--fgColor-default)',
          
          // Accent (Primary Actions - Blue)
          'accent-fg': 'var(--fgColor-accent)',
          'accent-emphasis': 'var(--bgColor-accent-emphasis)',
          'accent-subtle': 'var(--bgColor-accent-muted)',
          'accent-muted': 'var(--borderColor-accent-muted)',
          
          // Success (Green)
          'success-fg': 'var(--fgColor-success)',
          'success-emphasis': 'var(--bgColor-success-emphasis)',
          'success-subtle': 'var(--bgColor-success-muted)',
          'success-muted': 'var(--borderColor-success-muted)',
          
          // Danger (Red)
          'danger-fg': 'var(--fgColor-danger)',
          'danger-emphasis': 'var(--bgColor-danger-emphasis)',
          'danger-subtle': 'var(--bgColor-danger-muted)',
          'danger-muted': 'var(--borderColor-danger-muted)',
          
          // Attention (Yellow/Orange)
          'attention-fg': 'var(--fgColor-attention)',
          'attention-emphasis': 'var(--bgColor-attention-emphasis)',
          'attention-subtle': 'var(--bgColor-attention-muted)',
          'attention-muted': 'var(--borderColor-attention-muted)',
          
          // Done (Purple)
          'done-fg': 'var(--fgColor-done)',
          'done-emphasis': 'var(--bgColor-done-emphasis)',
          'done-subtle': 'var(--bgColor-done-muted)',
          'done-muted': 'var(--borderColor-done-muted)',
          
          // Sponsors (Pink)
          'sponsors-fg': 'var(--fgColor-sponsors)',
          'sponsors-emphasis': 'var(--bgColor-sponsors-emphasis)',
          'sponsors-subtle': 'var(--bgColor-sponsors-muted)',
          'sponsors-muted': 'var(--borderColor-sponsors-muted)',
        },
      },
      boxShadow: {
        'gh-card': '0 1px 0 rgba(31, 35, 40, 0.04)',
        'gh-focus': '0 0 0 3px rgba(48, 148, 255, 0.3)',
      },
      fontFamily: {
        // Extend Primer's font stacks with GitHub brand fonts
        // Using sans as default aligns with Primer's system
        'sans': ['Mona Sans', 'var(--fontStack-sansSerif)'],
        'mono': ['Monaspace Neon', 'var(--fontStack-monospace)'],
        // Legacy aliases for explicit usage
        'mona': ['Mona Sans', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
        'monaspace': ['Monaspace Neon', 'ui-monospace', 'SFMono-Regular', 'SF Mono', 'Menlo', 'Consolas', 'monospace'],
      },
      fontWeight: {
        // Map to Primer's base typography weights
        'light': 'var(--base-text-weight-light)', // 300
        'normal': 'var(--base-text-weight-normal)', // 400
        'medium': 'var(--base-text-weight-medium)', // 500
        'semibold': 'var(--base-text-weight-semibold)', // 600
      },
      fontSize: {
        // Map common sizes to Primer typography tokens
        'xs': 'var(--text-caption-size)', // 0.75rem = 12px
        'sm': 'var(--text-body-size-small)', // 0.75rem = 12px
        'base': 'var(--text-body-size-medium)', // 0.875rem = 14px
        'lg': 'var(--text-body-size-large)', // 1rem = 16px
        'xl': 'var(--text-title-size-small)', // 1rem = 16px
        '2xl': 'var(--text-title-size-medium)', // 1.25rem = 20px
        '3xl': 'var(--text-subtitle-size)', // 1.25rem = 20px
        '4xl': 'var(--text-title-size-large)', // 2rem = 32px
        '5xl': 'var(--text-display-size)', // 2.5rem = 40px
      },
      lineHeight: {
        // Primer line heights for proper alignment
        'tight': 'var(--text-caption-lineHeight)', // 1.3333
        'snug': 'var(--text-display-lineHeight)', // 1.4
        'normal': 'var(--text-body-lineHeight-medium)', // 1.42857
        'relaxed': 'var(--text-title-lineHeight-large)', // 1.5
        'loose': 'var(--text-subtitle-lineHeight)', // 1.6
      },
    },
  },
  plugins: [],
}

