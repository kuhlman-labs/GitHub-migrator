/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // GitHub Primer Brand Colors
        gh: {
          // Primary Palette - GitHub Green
          'green-1': '#BFFFD1',
          'green-3': '#5FED83',
          'green-4': '#08872B',
          'green-5': '#104C35',
          
          // Blue Palette - For accents and trust
          'blue-1': '#9EECFF',
          'blue-2': '#3094FF',
          'blue-4': '#0527FC',
          'blue-6': '#001C4D',
          
          // Purple Palette - For AI/Intelligence features
          'purple-1': '#D0B0FF',
          'purple-2': '#C06EFF',
          'purple-4': '#501DAF',
          'purple-6': '#000240',
          
          // Neutrals
          black: '#000000',
          white: '#FFFFFF',
          
          // Text colors (from Primer)
          'text-primary': '#1F2328',
          'text-secondary': '#656D76',
          'text-muted': '#8C959F',
          
          // Canvas colors
          'canvas-default': '#F6F8FA',
          'canvas-raised': '#FFFFFF',
          'canvas-inset': '#F6F8FA',
          
          // Borders
          'border-default': '#D1D9E0',
          'border-muted': '#E5E9ED',
          'border-hover': '#8C959F',
          
          // Header (darker)
          'header-bg': '#0D1117',
          'header-text': '#FFFFFF',
        },
      },
      boxShadow: {
        'gh-card': '0 1px 0 rgba(31, 35, 40, 0.04)',
        'gh-focus': '0 0 0 3px rgba(48, 148, 255, 0.3)',
      },
      fontFamily: {
        'mona': ['Mona Sans', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
        'monaspace': ['Monaspace Neon', 'ui-monospace', 'SFMono-Regular', 'SF Mono', 'Menlo', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
}

