import { Page, expect } from '@playwright/test';

export interface TestUser {
  email: string;
  password: string;
  role: string;
}

export const TEST_USERS = {
  admin: {
    email: 'user@plantd.local',
    password: 'User2024!',
    role: 'admin'
  },
  user: {
    email: 'user@plantd.local', 
    password: 'User2024!',
    role: 'user'
  },
  operator: {
    email: 'operator@plantd.local',
    password: 'Operator2024!', 
    role: 'operator'
  }
};

export async function loginAs(page: Page, user: TestUser): Promise<void> {
  await page.goto('/login');
  
  await page.fill('[name="email"]', user.email);
  await page.fill('[name="password"]', user.password);
  await page.click('button[type="submit"]');
  
  // Wait for redirect to dashboard
  await expect(page).toHaveURL('/dashboard');
  await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
}

export async function loginAsAdmin(page: Page): Promise<void> {
  await loginAs(page, TEST_USERS.admin);
}

export async function loginAsUser(page: Page): Promise<void> {
  await loginAs(page, TEST_USERS.user);
}

export async function loginAsOperator(page: Page): Promise<void> {
  await loginAs(page, TEST_USERS.operator);
}

export async function logout(page: Page): Promise<void> {
  await page.click('[data-testid="user-menu"]');
  await page.click('[data-testid="logout-button"]');
  
  // Wait for redirect to login
  await expect(page).toHaveURL('/login');
}

export async function expectAuthenticated(page: Page): Promise<void> {
  await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
}

export async function expectUnauthenticated(page: Page): Promise<void> {
  await expect(page).toHaveURL('/login');
} 
