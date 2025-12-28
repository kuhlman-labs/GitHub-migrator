/**
 * MSW server setup for Node.js environment (tests).
 */
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

// Create and export the server
export const server = setupServer(...handlers);

