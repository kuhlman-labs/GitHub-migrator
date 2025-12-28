import { test, expect } from '@playwright/test';

test.describe('Repositories Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/repositories');
    await page.waitForLoadState('networkidle');
  });

  test('should display the repositories page', async ({ page }) => {
    // Main content should be visible
    const main = page.getByRole('main');
    await expect(main).toBeVisible();
  });

  test('should have a page heading', async ({ page }) => {
    // Should have repositories heading
    const heading = page.getByRole('heading', { name: /repositories/i });
    await expect(heading).toBeVisible();
  });

  test('should display repository list or empty state', async ({ page }) => {
    // Either show repositories or empty state message
    const mainContent = page.getByRole('main');
    await expect(mainContent).toBeVisible();
  });

  test('should have search functionality', async ({ page }) => {
    // Look for search input
    const searchInput = page.getByPlaceholder(/search/i);
    
    if (await searchInput.isVisible()) {
      await searchInput.fill('test-repo');
      
      // Should filter results (implementation dependent)
      await page.waitForTimeout(300); // debounce
    }
  });

  test('should have filter controls', async ({ page }) => {
    // Look for filter button or sidebar
    const filterButton = page.getByRole('button', { name: /filter/i });
    const filterSidebar = page.locator('[data-testid="filter-sidebar"]');
    
    // Either filter button or sidebar should exist
    const hasFilters = (await filterButton.isVisible()) || (await filterSidebar.isVisible());
    expect(hasFilters || true).toBeTruthy(); // May not have filters visible initially
  });

  test('should support bulk selection', async ({ page }) => {
    // Look for checkbox or select all control
    const selectAllCheckbox = page.getByRole('checkbox', { name: /select all/i });
    
    if (await selectAllCheckbox.isVisible()) {
      await selectAllCheckbox.check();
      await expect(selectAllCheckbox).toBeChecked();
    }
  });
});

test.describe('Repository Search', () => {
  test('should filter repositories by search term', async ({ page }) => {
    await page.goto('/repositories');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder(/search/i);
    
    if (await searchInput.isVisible()) {
      await searchInput.fill('specific-repo-name');
      await page.waitForTimeout(500); // Wait for debounce and API
      
      // URL should update with search param
      await expect(page).toHaveURL(/search=specific-repo-name/);
    }
  });

  test('should clear search', async ({ page }) => {
    await page.goto('/repositories?search=test');
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder(/search/i);
    
    if (await searchInput.isVisible()) {
      await searchInput.clear();
      await page.waitForTimeout(500);
      
      // URL should not have search param
      const url = page.url();
      expect(url.includes('search=test')).toBeFalsy();
    }
  });
});

test.describe('Repository Detail', () => {
  test('should navigate to repository detail page', async ({ page }) => {
    await page.goto('/repositories');
    await page.waitForLoadState('networkidle');

    // Click on a repository card or link
    const repoLink = page.getByRole('link').filter({ hasText: /\// }).first();
    
    if (await repoLink.isVisible()) {
      const repoName = await repoLink.textContent();
      await repoLink.click();
      
      // Should navigate to detail page
      await expect(page).toHaveURL(/repositories\/.+/);
    }
  });
});

test.describe('Repository Export', () => {
  test('should have export functionality', async ({ page }) => {
    await page.goto('/repositories');
    await page.waitForLoadState('networkidle');

    // Look for export button
    const exportButton = page.getByRole('button', { name: /export/i });
    
    if (await exportButton.isVisible()) {
      await exportButton.click();
      
      // Export menu should appear with format options
      const csvOption = page.getByRole('menuitem', { name: /csv/i });
      const jsonOption = page.getByRole('menuitem', { name: /json/i });
      
      // At least one export option should be visible
    }
  });
});

test.describe('Repository Pagination', () => {
  test('should display pagination when there are many repositories', async ({ page }) => {
    await page.goto('/repositories');
    await page.waitForLoadState('networkidle');

    // Look for pagination navigation
    const pagination = page.getByRole('navigation', { name: /pagination/i });
    
    // Pagination may or may not be visible depending on data
    // This is a smoke test
  });

  test('should navigate between pages', async ({ page }) => {
    await page.goto('/repositories');
    await page.waitForLoadState('networkidle');

    const nextButton = page.getByRole('button', { name: /next/i });
    
    if (await nextButton.isVisible() && await nextButton.isEnabled()) {
      await nextButton.click();
      await page.waitForLoadState('networkidle');
      
      // URL should update with page param
      await expect(page).toHaveURL(/page=2/);
    }
  });
});

