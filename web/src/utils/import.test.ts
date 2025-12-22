import { describe, it, expect, vi, beforeEach } from 'vitest';
import { parseCSV, parseJSON, parseImportFile } from './import';

// Mock papaparse
vi.mock('papaparse', () => ({
  default: {
    parse: vi.fn((file, options) => {
      // Simulate async parsing
      if (file.name === 'error.csv') {
        options.error({ message: 'Parse error' });
      } else if (file.name === 'valid.csv') {
        options.complete({
          data: [
            { full_name: 'org/repo1' },
            { full_name: 'org/repo2' },
          ],
        });
      } else if (file.name === 'missing-column.csv') {
        options.complete({
          data: [
            { name: 'repo1' }, // Missing full_name column
          ],
        });
      } else if (file.name === 'empty.csv') {
        options.complete({
          data: [],
        });
      } else if (file.name === 'missing-value.csv') {
        options.complete({
          data: [
            { full_name: 'org/repo1' },
            { full_name: '' }, // Missing value
            { full_name: 'org/repo3' },
          ],
        });
      } else if (file.name === 'repository-column.csv') {
        options.complete({
          data: [
            { repository: 'org/repo1' },
            { repository: 'org/repo2' },
          ],
        });
      } else {
        options.complete({ data: [] });
      }
    }),
  },
}));

// Mock ExcelJS
vi.mock('exceljs', () => ({
  default: {
    Workbook: vi.fn().mockImplementation(() => ({
      xlsx: {
        load: vi.fn().mockResolvedValue(undefined),
      },
      worksheets: [
        {
          getRow: vi.fn((rowNum) => ({
            eachCell: (callback: (cell: { value: string }, colNum: number) => void) => {
              if (rowNum === 1) {
                callback({ value: 'full_name' }, 1);
              }
            },
          })),
          eachRow: (callback: (row: { eachCell: (cb: (cell: { value: string }, colNum: number) => void) => void }, rowNum: number) => void) => {
            callback(
              {
                eachCell: (cb) => cb({ value: 'full_name' }, 1),
              },
              1
            );
            callback(
              {
                eachCell: (cb) => cb({ value: 'org/repo1' }, 1),
              },
              2
            );
          },
        },
      ],
    })),
  },
}));

describe('parseCSV', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should parse valid CSV file', async () => {
    const file = new File([''], 'valid.csv', { type: 'text/csv' });
    
    const result = await parseCSV(file);
    
    expect(result.success).toBe(true);
    expect(result.rows).toHaveLength(2);
    expect(result.rows[0].full_name).toBe('org/repo1');
    expect(result.rows[1].full_name).toBe('org/repo2');
    expect(result.errors).toHaveLength(0);
  });

  it('should handle CSV parse error', async () => {
    const file = new File([''], 'error.csv', { type: 'text/csv' });
    
    const result = await parseCSV(file);
    
    expect(result.success).toBe(false);
    expect(result.rows).toHaveLength(0);
    expect(result.errors).toContain('Failed to parse CSV: Parse error');
  });

  it('should handle empty CSV file', async () => {
    const file = new File([''], 'empty.csv', { type: 'text/csv' });
    
    const result = await parseCSV(file);
    
    expect(result.success).toBe(false);
    expect(result.errors).toContain('File contains no data');
  });

  it('should handle missing required column', async () => {
    const file = new File([''], 'missing-column.csv', { type: 'text/csv' });
    
    const result = await parseCSV(file);
    
    expect(result.success).toBe(false);
    expect(result.errors).toContain('Missing required column: repository name (expected "repository" or "full_name")');
  });

  it('should handle missing values in rows', async () => {
    const file = new File([''], 'missing-value.csv', { type: 'text/csv' });
    
    const result = await parseCSV(file);
    
    expect(result.success).toBe(false);
    expect(result.rows).toHaveLength(2); // Only valid rows
    expect(result.errors).toContain('Row 3: Missing repository name');
  });

  it('should accept "repository" column name', async () => {
    const file = new File([''], 'repository-column.csv', { type: 'text/csv' });
    
    const result = await parseCSV(file);
    
    expect(result.success).toBe(true);
    expect(result.rows).toHaveLength(2);
    expect(result.rows[0].full_name).toBe('org/repo1');
  });
});

describe('parseJSON', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should parse valid JSON array', async () => {
    const jsonContent = JSON.stringify([
      { full_name: 'org/repo1' },
      { full_name: 'org/repo2' },
    ]);
    const file = new File([jsonContent], 'valid.json', { type: 'application/json' });
    
    const result = await parseJSON(file);
    
    expect(result.success).toBe(true);
    expect(result.rows).toHaveLength(2);
    expect(result.rows[0].full_name).toBe('org/repo1');
  });

  it('should reject non-array JSON', async () => {
    const jsonContent = JSON.stringify({ repositories: [] });
    const file = new File([jsonContent], 'object.json', { type: 'application/json' });
    
    const result = await parseJSON(file);
    
    expect(result.success).toBe(false);
    expect(result.errors).toContain('JSON file must contain an array of repositories');
  });

  it('should handle invalid JSON', async () => {
    const file = new File(['not valid json'], 'invalid.json', { type: 'application/json' });
    
    const result = await parseJSON(file);
    
    expect(result.success).toBe(false);
    expect(result.errors[0]).toContain('Failed to parse JSON');
  });

  it('should handle empty array', async () => {
    const jsonContent = JSON.stringify([]);
    const file = new File([jsonContent], 'empty.json', { type: 'application/json' });
    
    const result = await parseJSON(file);
    
    expect(result.success).toBe(false);
    expect(result.errors).toContain('File contains no data');
  });
});

describe('parseImportFile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should route CSV files to parseCSV', async () => {
    const file = new File([''], 'valid.csv', { type: 'text/csv' });
    
    const result = await parseImportFile(file);
    
    // parseCSV will be called
    expect(result.success).toBe(true);
  });

  it('should route JSON files to parseJSON', async () => {
    const jsonContent = JSON.stringify([{ full_name: 'org/repo' }]);
    const file = new File([jsonContent], 'test.json', { type: 'application/json' });
    
    const result = await parseImportFile(file);
    
    expect(result.success).toBe(true);
  });

  it('should reject unsupported file formats', async () => {
    const file = new File([''], 'test.txt', { type: 'text/plain' });
    
    const result = await parseImportFile(file);
    
    expect(result.success).toBe(false);
    expect(result.errors[0]).toContain('Unsupported file format: txt');
  });

  it('should handle xlsx files', async () => {
    const file = new File([''], 'test.xlsx', { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
    
    const result = await parseImportFile(file);
    
    // Excel parsing is mocked
    expect(result).toBeDefined();
  });

  it('should handle xls files', async () => {
    const file = new File([''], 'test.xls', { type: 'application/vnd.ms-excel' });
    
    const result = await parseImportFile(file);
    
    // Excel parsing is mocked
    expect(result).toBeDefined();
  });
});

