import { useAuthStore } from "../stores/auth";
import { useToastStore } from "../stores/toast";
import type { Router } from "vue-router";
import { useRouter } from "vue-router";

/**
 * Checks if an error is a network/connectivity error
 */
export const isNetworkError = (error: unknown): boolean => {
  if (error instanceof Error) {
    const message = error.message.toLowerCase();
    const name = (error as any).name?.toLowerCase() || "";

    // Check message content
    if (message.includes("failed to fetch") || message.includes("network request failed") || message.includes("fetch is not defined") || message.includes("cannot connect") || message.includes("networkerror") || message.includes("net::err") || message.includes("connection refused")) {
      return true;
    }

    // Check error name (DOMException sets this to 'NetworkError')
    if (name === "networkerror") {
      return true;
    }

    // Check for generic network-related message patterns
    if (message.includes("network") && (message.includes("error") || message.includes("failed"))) {
      return true;
    }

    return false;
  }

  return false;
};

/**
 * Checks if the error is due to server being unreachable
 */
export const isServerUnreachable = (error: unknown): boolean => {
  if (error instanceof Error) {
    const message = error.message.toLowerCase();
    return message.includes("websocket connection closed") || message.includes("reconnection failed") || message.includes("econnrefused") || message.includes("enotfound") || isNetworkError(error);
  }
  return isNetworkError(error);
};

/**
 * Logs out the user and shows an error message, then redirects to login
 * @param errorMessage The error message to display to the user
 * @param routerInstance Optional router instance to use for navigation (from component context)
 */
export const handleServerUnreachable = async (errorMessage = "Server is not reachable. Please try again later.", routerInstance?: Router) => {
  const authStore = useAuthStore();
  const toastStore = useToastStore();

  // Log the error for debugging
  console.error("[Server Unreachable]", errorMessage);

  // Show error toast before logout
  toastStore.error({
    title: "Connection Error",
    message: errorMessage,
  });

  // Logout the user
  authStore.logout();

  // Redirect to login page
  try {
    // Use provided router or try to get from component context
    const router = routerInstance || useRouter();
    // Use router.push instead of window.location to trigger proper cleanup
    await router.push("/login");
  } catch (error) {
    // If router not available, fallback to window.location
    console.warn("[Server Unreachable] Router unavailable, using window.location.assign");
    window.location.assign("/login");
  }
};

/**
 * Attempts to handle an error and returns true if it was a server unreachable error
 */
export const handlePotentialServerError = async (error: unknown): Promise<boolean> => {
  const isNetworkErr = isNetworkError(error);
  const isServerErr = isServerUnreachable(error);

  console.log("[handlePotentialServerError] Error analysis:", {
    isNetworkError: isNetworkErr,
    isServerUnreachable: isServerErr,
    errorMessage: error instanceof Error ? error.message : String(error),
    errorName: (error as any)?.name || "unknown",
  });

  if (isServerErr) {
    await handleServerUnreachable(error instanceof Error ? error.message : "Server is not reachable. Please try again later.");
    return true;
  }
  return false;
};
