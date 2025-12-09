// Jest setup file for GraphQL service tests

// Set test environment variables
process.env.NODE_ENV = 'test';
process.env.JWT_SECRET = 'test-jwt-secret-for-testing-only';

// Global test timeout
jest.setTimeout(10000);

// Mock console.error to reduce noise in tests (optional)
// Uncomment if you want to suppress expected errors in tests
// const originalConsoleError = console.error;
// beforeAll(() => {
//   console.error = jest.fn();
// });
// afterAll(() => {
//   console.error = originalConsoleError;
// });
