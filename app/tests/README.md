# App Service Testing

This directory contains end-to-end tests for the plantd App Service using Playwright.

## Setup

The testing framework is already configured. To run tests:

```bash
# Install dependencies (if not already done)
bun install

# Install Playwright browsers
bun playwright install

# Run all tests
bun run test:e2e

# Run tests in headed mode (see browser)
bun run test:e2e:headed

# Run tests in debug mode
bun run test:e2e:debug

# Open Playwright UI for interactive testing
bun run test:e2e:ui

# View test report
bun run test:e2e:report
```

## Test Structure

- `e2e/auth/` - Authentication flow tests
- `e2e/dashboard/` - Dashboard functionality tests  
- `e2e/security/` - Security-related tests (CSRF, XSS, etc.)
- `utils/` - Test helper functions and utilities

## Configuration

- `playwright.config.ts` - Main Playwright configuration
- Tests run against `https://127.0.0.1:8443` (the app service)
- Tests accept self-signed certificates for development
- Cross-browser testing: Chrome, Firefox, Safari, Mobile

## Notes

- Tests expect the app service to be running via `overmind start app`
- The webServer configuration will automatically start the app if not running
- Tests are designed to be resilient and handle missing UI elements gracefully
- Authentication tests use placeholder credentials that match the current auth setup 
