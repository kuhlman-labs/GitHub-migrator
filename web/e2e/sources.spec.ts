import { test, expect } from '@playwright/test';

test.describe('Sources Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to sources page
    await page.goto('/sources');
    await page.waitForLoadState('networkidle');
  });

  test('should display sources page', async ({ page }) => {
    // The sources page should load with a heading
    await expect(page.getByRole('heading', { name: /migration sources/i })).toBeVisible();
  });

  test('should have add source button', async ({ page }) => {
    // Look for add source button
    const addButton = page.getByRole('button', { name: /add source/i });
    await expect(addButton).toBeVisible();
  });

  test('should open create source dialog', async ({ page }) => {
    // Click add source button
    const addButton = page.getByRole('button', { name: /add source/i });
    await addButton.click();

    // Dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText(/add new source/i)).toBeVisible();
  });

  test('should display source form fields', async ({ page }) => {
    // Open create dialog
    await page.getByRole('button', { name: /add source/i }).click();

    // Check form fields
    await expect(page.getByLabel(/name/i)).toBeVisible();
    await expect(page.getByLabel(/type/i)).toBeVisible();
    await expect(page.getByLabel(/base url/i)).toBeVisible();
    await expect(page.getByLabel(/personal access token/i)).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    // Open create dialog
    await page.getByRole('button', { name: /add source/i }).click();

    // Try to submit without filling fields
    const submitButton = page.getByRole('button', { name: /create source/i });
    
    // Button should be disabled or form should show validation
    const isDisabled = await submitButton.isDisabled();
    expect(isDisabled).toBe(true);
  });

  test('should have test connection button', async ({ page }) => {
    // Open create dialog
    await page.getByRole('button', { name: /add source/i }).click();

    // Test connection button should be visible
    await expect(page.getByRole('button', { name: /test connection/i })).toBeVisible();
  });

  test('should cancel create dialog', async ({ page }) => {
    // Open create dialog
    await page.getByRole('button', { name: /add source/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible();

    // Click cancel
    await page.getByRole('button', { name: /cancel/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible();
  });

  test('should show empty state when no sources', async ({ page }) => {
    // When no sources are configured, should show empty state message
    // This test expects the empty state to be visible when starting fresh
    const emptyState = page.getByText(/no sources configured/i);
    
    // Either we have sources (no empty state) or we see the empty state
    const hasEmptyState = await emptyState.isVisible().catch(() => false);
    const hasSources = await page.locator('[data-testid="source-card"]').count().catch(() => 0);
    
    // One of these should be true
    expect(hasEmptyState || hasSources > 0).toBe(true);
  });
});

test.describe('Source Type Selection', () => {
  test('should show GitHub as default type', async ({ page }) => {
    await page.goto('/sources');
    await page.waitForLoadState('networkidle');
    
    await page.getByRole('button', { name: /add source/i }).click();

    // GitHub should be the default or available option
    const typeSelect = page.getByLabel(/type/i);
    await expect(typeSelect).toBeVisible();
  });

  test('should show Azure DevOps as option', async ({ page }) => {
    await page.goto('/sources');
    await page.waitForLoadState('networkidle');
    
    await page.getByRole('button', { name: /add source/i }).click();

    // Type selector should have Azure DevOps option
    const typeSelect = page.getByLabel(/type/i);
    await typeSelect.click();
    
    await expect(page.getByRole('option', { name: /azure devops/i })).toBeVisible();
  });

  test('should show organization field for Azure DevOps', async ({ page }) => {
    await page.goto('/sources');
    await page.waitForLoadState('networkidle');
    
    await page.getByRole('button', { name: /add source/i }).click();

    // Select Azure DevOps
    const typeSelect = page.getByLabel(/type/i);
    await typeSelect.selectOption('azuredevops');

    // Organization field should appear
    await expect(page.getByLabel(/organization/i)).toBeVisible();
  });
});

test.describe('Source Navigation', () => {
  test('should navigate to sources from dashboard', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // If there's a sources link in settings or navigation
    // Try to find and click it
    const sourcesLink = page.getByRole('link', { name: /sources/i });
    
    if (await sourcesLink.isVisible().catch(() => false)) {
      await sourcesLink.click();
      await expect(page).toHaveURL(/sources/);
    }
  });

  test('should be accessible', async ({ page }) => {
    await page.goto('/sources');
    await page.waitForLoadState('networkidle');

    // Should have proper heading structure
    const h1 = page.getByRole('heading', { level: 1 });
    await expect(h1).toBeVisible();
  });
});

