/**
 * Sanitization utilities for user input
 */

/**
 * Sanitizes a string by escaping HTML special characters
 * Prevents XSS attacks by converting potentially dangerous characters
 */
export function sanitizeHtml(input: string): string {
  const div = document.createElement("div");
  div.textContent = input;
  return div.innerHTML;
}

/**
 * Sanitizes a URL to prevent javascript: and data: URIs
 * Only allows http:, https:, and relative URLs
 */
export function sanitizeUrl(url: string): string {
  const trimmed = url.trim();
  const lower = trimmed.toLowerCase();

  // Block javascript: and data: URIs
  if (lower.startsWith("javascript:") || lower.startsWith("data:") || lower.startsWith("vbscript:")) {
    return "";
  }

  // Allow relative URLs, http, and https
  if (trimmed.startsWith("/") || trimmed.startsWith("./") || trimmed.startsWith("../") || lower.startsWith("http://") || lower.startsWith("https://")) {
    return trimmed;
  }

  // If it doesn't match any safe pattern, return empty
  return "";
}

/**
 * Sanitizes a string for use in display names
 * Removes control characters and limits length
 */
export function sanitizeDisplayName(input: string, maxLength = 100): string {
  // Remove control characters (ASCII 0-31 except tab, newline, carriage return)
  let sanitized = input.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // Trim whitespace
  sanitized = sanitized.trim();

  // Limit length
  if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength);
  }

  return sanitized;
}

/**
 * Sanitizes a channel name (alphanumeric, hyphens, underscores only)
 */
export function sanitizeChannelName(input: string): string {
  // Only allow alphanumeric, hyphens, and underscores
  return input.replace(/[^a-zA-Z0-9_-]/g, "").trim();
}

/**
 * Sanitizes a numeric input, ensuring it's a valid number within bounds
 */
export function sanitizeNumber(input: unknown, min?: number, max?: number, defaultValue = 0): number {
  let num = typeof input === "number" ? input : parseFloat(String(input));

  if (isNaN(num)) {
    return defaultValue;
  }

  if (min !== undefined && num < min) {
    num = min;
  }

  if (max !== undefined && num > max) {
    num = max;
  }

  return num;
}

/**
 * Sanitizes a tag string (used for video tags)
 * Splits by comma, trims each tag, removes empty tags
 */
export function sanitizeTags(input: string): string[] {
  return input
    .split(",")
    .map((tag) => sanitizeDisplayName(tag, 50))
    .filter((tag) => tag.length > 0);
}

/**
 * Validates and sanitizes an email address
 */
export function sanitizeEmail(input: string): string {
  const trimmed = input.trim().toLowerCase();
  // Basic email validation regex
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  if (!emailRegex.test(trimmed)) {
    return "";
  }

  return trimmed;
}
