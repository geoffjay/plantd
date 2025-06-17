import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test('should successfully log in with valid credentials', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    
    // Verify login form is visible
    await expect(page.locator('form')).toBeVisible();
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    
    // Fill in login credentials (using placeholder credentials)
    await page.fill('input[name="email"]', 'admin@plantd.local');
    await page.fill('input[name="password"]', 'password123');
    
    // Submit the form
    await page.click('button[type="submit"]');
    
    // Should redirect to dashboard or home page
    await page.waitForURL((url) => url.pathname === '/dashboard' || url.pathname === '/');
    
    // Verify successful login by checking for authenticated content
    // This will depend on what appears after successful login
    await expect(page).toHaveURL(/\/(dashboard|$)/);
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.goto('/login');
    
    await page.fill('input[name="email"]', 'invalid@example.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');
    
    // Should stay on login page and show error
    await expect(page).toHaveURL(/\/login/);
    // Look for error message (this will depend on implementation)
    await expect(page.locator('text=Invalid')).toBeVisible();
  });

  test('should handle CSRF token correctly', async ({ page }) => {
    await page.goto('/login');
    
    // Verify CSRF token is present in the form (may be hidden)
    const csrfInput = page.locator('input[name="csrf"]');
    await expect(csrfInput).toBeAttached(); // Just check it exists
    
    const csrfValue = await csrfInput.getAttribute('value');
    expect(csrfValue).toBeTruthy();
    expect(csrfValue).toHaveLength(36); // UUID format
  });
}); 
