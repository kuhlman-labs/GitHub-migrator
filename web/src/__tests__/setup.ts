/**
 * Test setup file for Vitest.
 * Configures testing environment with DOM mocking and testing utilities.
 */
import '@testing-library/jest-dom';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { cleanup } from '@testing-library/react';
import { server } from './mocks/server';

// Cleanup after each test case (e.g., clearing jsdom)
afterEach(() => {
  cleanup();
});

// Start MSW server before all tests
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'warn' });
});

// Reset handlers after each test
afterEach(() => {
  server.resetHandlers();
});

// Clean up after all tests are done
afterAll(() => {
  server.close();
});

// Mock window.matchMedia for components that use it
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// Mock IntersectionObserver for components that use it
class MockIntersectionObserver {
  observe = () => null;
  disconnect = () => null;
  unobserve = () => null;
}

Object.defineProperty(window, 'IntersectionObserver', {
  writable: true,
  value: MockIntersectionObserver,
});

// Mock ResizeObserver for components that use it
class MockResizeObserver {
  observe = () => null;
  disconnect = () => null;
  unobserve = () => null;
}

Object.defineProperty(window, 'ResizeObserver', {
  writable: true,
  value: MockResizeObserver,
});

// Mock adoptedStyleSheets for Primer React tooltip polyfill
Object.defineProperty(document, 'adoptedStyleSheets', {
  writable: true,
  value: [],
});

// Mock CSSStyleSheet for popover polyfill
class MockCSSStyleSheet {
  replaceSync = () => {};
  cssRules = [];
  insertRule = () => 0;
  deleteRule = () => {};
}

// @ts-expect-error - mocking CSSStyleSheet for jsdom
globalThis.CSSStyleSheet = MockCSSStyleSheet;

