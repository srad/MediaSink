import { onMounted, onUnmounted, ref } from "vue";

/**
 * Composable for periodic service worker update checking
 *
 * This composable periodically checks for new service worker updates
 * by asking the browser to re-check for SW updates at regular intervals.
 *
 * When a new SW is detected and becomes active:
 * - The controllerchange event fires
 * - Components listening to this event (e.g., App.vue) show update prompts
 *
 * @param intervalMs - Check interval in milliseconds (default: 60000 = 1 minute)
 * @returns Reactive state indicating if check is in progress
 */
export const useSwUpdateCheck = (intervalMs: number = 60000) => {
  const isChecking = ref(false);
  let checkInterval: number | null = null;

  const checkForUpdates = async () => {
    if (!("serviceWorker" in navigator)) {
      return; // Service workers not supported
    }

    try {
      isChecking.value = true;
      // This tells the browser to:
      // 1. Fetch the service worker script again
      // 2. If it's different, install the new SW
      // 3. If skipWaiting is enabled, activate it immediately
      // 4. Trigger controllerchange event on the page
      await navigator.serviceWorker.getRegistration().then((registration) => {
        if (registration) {
          // Force a re-check for new SW
          registration.update();
        }
      });
    } catch (error) {
      console.error("[SW Update Check] Error checking for updates:", error);
    } finally {
      isChecking.value = false;
    }
  };

  const startPeriodicCheck = () => {
    // Check immediately on mount
    checkForUpdates();

    // Then check periodically
    checkInterval = window.setInterval(() => {
      checkForUpdates();
    }, intervalMs);

    console.log(`[SW Update Check] Started periodic checks every ${intervalMs}ms`);
  };

  const stopPeriodicCheck = () => {
    if (checkInterval !== null) {
      clearInterval(checkInterval);
      checkInterval = null;
      console.log("[SW Update Check] Stopped periodic checks");
    }
  };

  onMounted(() => {
    startPeriodicCheck();
  });

  onUnmounted(() => {
    stopPeriodicCheck();
  });

  return {
    isChecking,
    checkForUpdates,
    startPeriodicCheck,
    stopPeriodicCheck,
  };
};
