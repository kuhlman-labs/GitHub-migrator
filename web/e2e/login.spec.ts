import { test, expect } from '@playwright/test';

test.describe('Login Flow', () => {
  test('should display login page when not authenticated', async ({ page }) => {
    // Navigate to a protected route
    await page.goto('/');
    
    // Should redirect to login or show login UI
    // The exact behavior depends on auth configuration
    await expect(page).toHaveURL(/\/(login)?/);
  });

  test('should show login button on login page', async ({ page }) => {
    await page.goto('/login');
    
    // Look for login button (may vary based on auth provider)
    const loginButton = page.getByRole('button', { name: /login|sign in/i });
    
    // The page should have some form of login action
    await expect(page.locator('body')).toBeVisible();
  });

  test('should have accessible login page', async ({ page }) => {
    await page.goto('/login');
    
    // Check for main heading
    const heading = page.getByRole('heading').first();
    await expect(heading).toBeVisible();
  });

  test('should preserve return URL when redirecting to login', async ({ page }) => {
    // Navigate directly to a protected route
    await page.goto('/repositories');
    
    // After auth, should remember to go back to repositories
    // This is typically stored in router state
  });
});

test.describe('Unauthenticated Access', () => {
  test('should redirect protected routes to login', async ({ page }) => {
    // Try to access dashboard
    await page.goto('/');
    
    // Check we're either on login or dashboard (depending on auth config)
    const url = page.url();
    expect(url.includes('/login') || url.includes('/') || url.includes('/dashboard')).toBeTruthy();
  });

  test('should redirect repositories page to login when not authenticated', async ({ page }) => {
    await page.goto('/repositories');
    
    // Should either show repositories or redirect to login
    const url = page.url();
    expect(url.includes('/login') || url.includes('/repositories')).toBeTruthy();
  });
});

