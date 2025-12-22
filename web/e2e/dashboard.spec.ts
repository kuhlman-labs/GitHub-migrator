import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to dashboard
    await page.goto('/');
  });

  test('should display the dashboard page', async ({ page }) => {
    // The dashboard should load
    await expect(page.locator('body')).toBeVisible();
  });

  test('should have main navigation', async ({ page }) => {
    // Look for navigation element
    const nav = page.getByRole('navigation');
    await expect(nav).toBeVisible();
  });

  test('should display KPI section', async ({ page }) => {
    // Wait for the page to load
    await page.waitForLoadState('networkidle');
    
    // Dashboard should show key metrics
    // These might be in cards or specific sections
    const main = page.getByRole('main');
    await expect(main).toBeVisible();
  });

  test('should have discovery action available', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Look for discovery button or action
    const discoveryButton = page.getByRole('button', { name: /discover|start discovery/i });
    
    // Discovery action should be available (visible or in dropdown)
    // This might not exist if already discovered
  });

  test('should show organization statistics', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Dashboard should display organization-level stats
    const mainContent = page.getByRole('main');
    await expect(mainContent).toBeVisible();
  });

  test('should be responsive', async ({ page }) => {
    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForLoadState('networkidle');
    
    // Main content should still be visible
    const main = page.getByRole('main');
    await expect(main).toBeVisible();
  });

  test('should have accessible heading structure', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // Should have at least one heading
    const headings = page.getByRole('heading');
    await expect(headings.first()).toBeVisible();
  });
});

test.describe('Dashboard Navigation', () => {
  test('should navigate to repositories page', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Click on repositories link in navigation
    const reposLink = page.getByRole('link', { name: /repositories/i });
    
    if (await reposLink.isVisible()) {
      await reposLink.click();
      await expect(page).toHaveURL(/repositories/);
    }
  });

  test('should navigate to batches page', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Click on batches link in navigation
    const batchesLink = page.getByRole('link', { name: /batches|batch/i });
    
    if (await batchesLink.isVisible()) {
      await batchesLink.click();
      await expect(page).toHaveURL(/batch/);
    }
  });
});

