import Papa from 'papaparse';
import ExcelJS from 'exceljs';

export interface ParsedImportRow {
  full_name: string;
}

export interface ImportParseResult {
  success: boolean;
  rows: ParsedImportRow[];
  errors: string[];
}

/**
 * Parse a CSV file and extract repository names and migration settings
 */
export async function parseCSV(file: File): Promise<ImportParseResult> {
  return new Promise((resolve) => {
    Papa.parse(file, {
      header: true,
      skipEmptyLines: true,
      complete: (results) => {
        const { rows, errors } = processRows(results.data as Record<string, unknown>[]);
        resolve({
          success: errors.length === 0,
          rows,
          errors,
        });
      },
      error: (error) => {
        resolve({
          success: false,
          rows: [],
          errors: [`Failed to parse CSV: ${error.message}`],
        });
      },
    });
  });
}

/**
 * Parse an Excel file and extract repository names and migration settings
 */
export async function parseExcel(file: File): Promise<ImportParseResult> {
  try {
    const arrayBuffer = await file.arrayBuffer();
    const workbook = new ExcelJS.Workbook();
    await workbook.xlsx.load(arrayBuffer);

    // Get the first worksheet
    const worksheet = workbook.worksheets[0];

    if (!worksheet) {
      return {
        success: false,
        rows: [],
        errors: ['No worksheet found in Excel file'],
      };
    }

    // Convert to JSON
    const jsonData: Record<string, unknown>[] = [];
    const headers: string[] = [];

    // Get headers from first row
    worksheet.getRow(1).eachCell((cell, colNumber) => {
      headers[colNumber - 1] = String(cell.value || '').trim();
    });

    // Get data rows
    worksheet.eachRow((row, rowNumber) => {
      if (rowNumber === 1) return; // Skip header row

      const rowData: Record<string, unknown> = {};
      row.eachCell((cell, colNumber) => {
        const header = headers[colNumber - 1];
        if (header) {
          rowData[header] = cell.value;
        }
      });

      if (Object.keys(rowData).length > 0) {
        jsonData.push(rowData);
      }
    });

    const { rows, errors } = processRows(jsonData);
    return {
      success: errors.length === 0,
      rows,
      errors,
    };
  } catch (error: unknown) {
    return {
      success: false,
      rows: [],
      errors: [`Failed to parse Excel: ${error instanceof Error ? error.message : 'Unknown error'}`],
    };
  }
}

/**
 * Parse a JSON file and extract repository names and migration settings
 */
export async function parseJSON(file: File): Promise<ImportParseResult> {
  return new Promise((resolve) => {
    const reader = new FileReader();

    reader.onload = (e) => {
      try {
        const jsonData = JSON.parse(e.target?.result as string);

        if (!Array.isArray(jsonData)) {
          resolve({
            success: false,
            rows: [],
            errors: ['JSON file must contain an array of repositories'],
          });
          return;
        }

        const { rows, errors } = processRows(jsonData);
        resolve({
          success: errors.length === 0,
          rows,
          errors,
        });
      } catch (error: unknown) {
        resolve({
          success: false,
          rows: [],
          errors: [`Failed to parse JSON: ${error instanceof Error ? error.message : 'Unknown error'}`],
        });
      }
    };

    reader.onerror = () => {
      resolve({
        success: false,
        rows: [],
        errors: ['Failed to read JSON file'],
      });
    };

    reader.readAsText(file);
  });
}

/**
 * Process rows from parsed file data
 * Only extracts repository names - migration settings come from batch configuration
 */
function processRows(data: Record<string, unknown>[]): { rows: ParsedImportRow[]; errors: string[] } {
  const rows: ParsedImportRow[] = [];
  const errors: string[] = [];

  if (data.length === 0) {
    errors.push('File contains no data');
    return { rows, errors };
  }

  // Check for required column (try different possible column names)
  const firstRow = data[0];
  const possibleNames = ['full_name', 'repository', 'Repository', 'Full_Name', 'full name', 'Repository Name'];
  const hasRepoColumn = possibleNames.some(name => name in firstRow);
  
  if (!hasRepoColumn) {
    errors.push('Missing required column: repository name (expected "repository" or "full_name")');
    return { rows, errors };
  }

  data.forEach((row, index) => {
    const lineNumber = index + 2; // +2 because index is 0-based and we skip header

    // Get repository name (try different case variations and column names)
    const full_name = row.repository || row.Repository || row.full_name || row.Full_Name || row['full name'] || row['Repository Name'] || '';

    if (!full_name || typeof full_name !== 'string' || !full_name.trim()) {
      errors.push(`Row ${lineNumber}: Missing repository name`);
      return;
    }

    rows.push({
      full_name: full_name.trim(),
    });
  });

  return { rows, errors };
}

/**
 * Auto-detect file format and parse accordingly
 */
export async function parseImportFile(file: File): Promise<ImportParseResult> {
  const extension = file.name.split('.').pop()?.toLowerCase();

  switch (extension) {
    case 'csv':
      return parseCSV(file);
    case 'xlsx':
    case 'xls':
      return parseExcel(file);
    case 'json':
      return parseJSON(file);
    default:
      return {
        success: false,
        rows: [],
        errors: [`Unsupported file format: ${extension}. Please use CSV, Excel (.xlsx), or JSON.`],
      };
  }
}

