export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes';
  
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

export function formatDuration(seconds: number): string {
  // Round to nearest second
  const roundedSeconds = Math.round(seconds);
  
  if (roundedSeconds < 60) return `${roundedSeconds}s`;
  if (roundedSeconds < 3600) {
    const mins = Math.floor(roundedSeconds / 60);
    const secs = roundedSeconds % 60;
    return `${mins}m ${secs}s`;
  }
  
  const hours = Math.floor(roundedSeconds / 3600);
  const minutes = Math.floor((roundedSeconds % 3600) / 60);
  return `${hours}h ${minutes}m`;
}

export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleString();
}

export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  if (diffDay < 30) return `${diffDay}d ago`;
  
  // For older dates, show months/years
  const diffMonth = Math.floor(diffDay / 30);
  const diffYear = Math.floor(diffDay / 365);
  
  if (diffMonth < 12) return `${diffMonth}mo ago`;
  return `${diffYear}y ago`;
}

export function isStaleTimestamp(dateString: string, daysThreshold: number = 30): boolean {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  return diffDays > daysThreshold;
}

export function formatTimestampWithStaleness(dateString: string, daysThreshold: number = 30): { 
  formatted: string; 
  isStale: boolean;
  fullDate: string;
} {
  return {
    formatted: formatRelativeTime(dateString),
    isStale: isStaleTimestamp(dateString, daysThreshold),
    fullDate: formatDate(dateString)
  };
}

/**
 * Formats a date string for use in datetime-local inputs.
 * Converts to local timezone and returns format: YYYY-MM-DDTHH:MM
 * 
 * This is needed because .toISOString() converts to UTC, but datetime-local
 * inputs interpret the value as local time, causing timezone mismatches.
 */
export function formatDateForInput(dateString: string | null | undefined): string {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  
  // Get local time components
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

/**
 * Sanitizes a repository name for GitHub compatibility.
 * GitHub repo names can only contain ASCII letters, digits, and ., -, _
 * Replaces slashes and spaces with hyphens.
 */
export function sanitizeRepoName(name: string): string {
  return name.replace(/[/ ]/g, '-');
}

/**
 * Computes the destination repository name for batch-level defaults.
 * 
 * For ADO repos: uses project-repo pattern (e.g., "DevOps-Terraform")
 * to preserve project context and avoid naming conflicts.
 * 
 * For GitHub repos: uses just the repo name.
 * 
 * @param fullName The full name of the repository (e.g., "org/project/repo" for ADO, "org/repo" for GitHub)
 * @param adoProject The ADO project name if this is an ADO repository, undefined otherwise
 * @returns The sanitized repository name suitable for GitHub
 */
export function getDestinationRepoName(fullName: string, adoProject?: string): string {
  const parts = fullName.split('/');
  
  // For ADO repos (org/project/repo format), use project-repo pattern
  if (adoProject && parts.length >= 3) {
    const project = sanitizeRepoName(parts[1]);
    const repo = sanitizeRepoName(parts[parts.length - 1]);
    return `${project}-${repo}`;
  }
  
  // For GitHub repos (org/repo format), use just the repo name
  if (parts.length >= 2) {
    return sanitizeRepoName(parts[parts.length - 1]);
  }
  
  // Fallback: sanitize the full name
  return sanitizeRepoName(fullName);
}

/**
 * Computes the full destination path for a repository when a batch destination org is set.
 * 
 * @param destOrg The destination organization
 * @param fullName The full name of the source repository
 * @param adoProject The ADO project name if this is an ADO repository
 * @returns The full destination path (e.g., "dest-org/DevOps-Terraform")
 */
export function computeBatchDefaultDestination(
  destOrg: string,
  fullName: string,
  adoProject?: string
): string {
  const repoName = getDestinationRepoName(fullName, adoProject);
  return `${destOrg}/${repoName}`;
}

