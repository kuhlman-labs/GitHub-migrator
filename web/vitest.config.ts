/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/__tests__/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules', 'dist', 'e2e'],
    css: {
      modules: {
        classNameStrategy: 'non-scoped',
      },
    },
    server: {
      deps: {
        inline: ['@primer/react'],
      },
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/__tests__/**',
        'src/main.tsx',
        'src/vite-env.d.ts',
        'src/App.tsx', // Main app entry - difficult to unit test
        'src/types/*.ts', // Pure type definitions
        'src/services/index.ts', // Re-export file
        'src/services/api.ts', // Re-export file
        'src/services/api/index.ts', // Re-export file
        'src/components/*/index.tsx', // Re-export files that just export other components
        'src/components/*/index.ts', // Re-export files
        'src/components/BatchManagement/BatchBuilder.tsx', // Complex component with extensive state management
        'src/components/Setup/SetupWizard.tsx', // Multi-step wizard with complex form state
      ],
      // Coverage thresholds - current achievement
      thresholds: {
        global: {
          statements: 65,
          branches: 60,
          functions: 65,
          lines: 70,
        },
      },
    },
  },
});

