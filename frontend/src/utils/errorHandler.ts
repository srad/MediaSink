import { useToastStore } from "@/stores/toast";

export interface ErrorDetails {
  title?: string;
  message: string;
  error?: unknown;
  logToConsole?: boolean;
}

/**
 * Centralized error handler that displays user-friendly error messages
 * and optionally logs errors to the console
 */
export function handleError(details: ErrorDetails): void {
  const toastStore = useToastStore();

  const title = details.title || "Error";
  let message = details.message;

  // Try to extract error message from different error types
  if (details.error) {
    if (details.error instanceof Error) {
      message = details.error.message || message;
    } else if (typeof details.error === "object" && details.error !== null) {
      // Handle API error responses
      const errorObj = details.error as Record<string, unknown>;
      if (errorObj.error && typeof errorObj.error === "string") {
        message = errorObj.error;
      } else if (errorObj.message && typeof errorObj.message === "string") {
        message = errorObj.message;
      }
    } else if (typeof details.error === "string") {
      message = details.error;
    }
  }

  // Display error toast
  toastStore.error({ title, message });

  // Log to console if requested (default: true)
  if (details.logToConsole !== false) {
    console.error(`[${title}]`, message, details.error);
  }
}

/**
 * Handle API errors specifically
 */
export function handleApiError(error: unknown, context?: string): void {
  handleError({
    title: context ? `${context} Failed` : "API Error",
    message: "An error occurred while communicating with the server",
    error,
    logToConsole: true,
  });
}

/**
 * Handle validation errors
 */
export function handleValidationError(message: string): void {
  const toastStore = useToastStore();
  toastStore.warn({
    title: "Validation Error",
    message,
  });
}

/**
 * Create a safe error message from unknown error types
 */
export function extractErrorMessage(error: unknown, fallback = "An unknown error occurred"): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  if (typeof error === "object" && error !== null) {
    const errorObj = error as Record<string, unknown>;
    if (errorObj.error && typeof errorObj.error === "string") {
      return errorObj.error;
    }
    if (errorObj.message && typeof errorObj.message === "string") {
      return errorObj.message;
    }
  }
  return fallback;
}
