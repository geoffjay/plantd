import { test, expect } from '@playwright/test';
import { loginAsAdmin, expectAuthenticated } from '../../utils/auth-helpers';

test.describe('Dashboard Overview', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should display main dashboard elements', async ({ page }) => {
    // Verify we're on the dashboard
    await expect(page).toHaveURL('/dashboard');
    await expectAuthenticated(page);
    
    // Check for main dashboard components
    await expect(page.locator('h1')).toContainText('Dashboard');
    
    // Check for metric cards (if they exist)
    const metricCards = page.locator('[data-testid*="metric-"]');
    if (await metricCards.count() > 0) {
      await expect(metricCards.first()).toBeVisible();
    }
    
    // Check for navigation elements
    const navigation = page.locator('[data-testid="main-nav"]');
    if (await navigation.count() > 0) {
      await expect(navigation).toBeVisible();
    }
  });

  test('should have working navigation links', async ({ page }) => {
    // First check if sidebar is hidden and needs to be opened
    const sidebar = page.locator('#sidenav');
    const isHidden = await sidebar.getAttribute('class');
    
    if (isHidden?.includes('hidden')) {
      // Click hamburger menu to open sidebar
      const menuButton = page.locator('i[class*="fa-bars"]').first();
      if (await menuButton.count() > 0) {
        await menuButton.click();
        await page.waitForTimeout(300); // Wait for animation
      }
    }
    
    // Test navigation to services if link exists
    const servicesLink = page.locator('a[href="/services"]');
    if (await servicesLink.count() > 0) {
      await servicesLink.click();
      await expect(page).toHaveURL('/services');
      
      // Navigate back to dashboard
      const dashboardLink = page.locator('a[href="/dashboard"]');
      if (await dashboardLink.count() > 0) {
        await dashboardLink.click();
        await expect(page).toHaveURL('/dashboard');
      }
    }
  });

  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    // Verify dashboard still works on mobile
    await expect(page.locator('h1')).toBeVisible();
    
    // Check if mobile menu exists and works
    const mobileMenuButton = page.locator('[data-testid="mobile-menu-button"]');
    if (await mobileMenuButton.count() > 0) {
      await mobileMenuButton.click();
      const mobileMenu = page.locator('[data-testid="mobile-menu"]');
      await expect(mobileMenu).toBeVisible();
    }
  });

  test('should handle real-time updates gracefully', async ({ page }) => {
    // Wait a moment to see if any real-time updates cause issues
    await page.waitForTimeout(2000);
    
    // Ensure page is still functional after potential updates
    await expect(page.locator('h1')).toBeVisible();
    await expectAuthenticated(page);
  });
}); 
