/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // GitHub color palette
        gh: {
          // Primary
          blue: '#0969DA',
          'blue-hover': '#0860CA',
          
          // Text
          'text-primary': '#1F2328',
          'text-secondary': '#656D76',
          'text-muted': '#8C959F',
          
          // Canvas
          'canvas-default': '#F6F8FA',
          'canvas-raised': '#FFFFFF',
          'canvas-inset': '#F6F8FA',
          
          // Borders
          'border-default': '#D1D9E0',
          'border-muted': '#E5E9ED',
          'border-hover': '#8C959F',
          
          // Header (darker like GitHub)
          'header-bg': '#0D1117',
          'header-text': '#FFFFFF',
          
          // Status colors
          success: '#1A7F37',
          'success-hover': '#1A7F37',
          'success-emphasis': '#2DA44E',
          danger: '#D1242F',
          'danger-hover': '#B52324',
          warning: '#9A6700',
          'warning-emphasis': '#BF8700',
          
          // State backgrounds
          'success-bg': '#DFF7E9',
          'danger-bg': '#FFEBEC',
          'warning-bg': '#FFF8C5',
          'info-bg': '#DBF0FF',
          'neutral-bg': '#F6F8FA',
        },
      },
      boxShadow: {
        'gh-card': '0 1px 0 rgba(31, 35, 40, 0.04)',
        'gh-focus': '0 0 0 3px rgba(9, 105, 218, 0.3)',
      },
    },
  },
  plugins: [],
}

