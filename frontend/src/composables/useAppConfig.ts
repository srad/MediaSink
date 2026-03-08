import { readonly, ref } from "vue";

declare global {
  interface Window {
    APP_NAME: string;
    APP_BASE: string;
    APP_SOCKETURL: string;
    APP_API_VERSION: string;
  }
}

interface AppConfig {
  appName: string;
  baseUrl: string;
  socketUrl: string;
  apiVersion: string;
}

// Singleton configuration
const config = ref<AppConfig>({
  appName: "",
  baseUrl: "",
  socketUrl: "",
  apiVersion: "",
});

let initialized = false;

/**
 * Initialize app configuration from window object
 * Should be called once during app setup
 */
export function initializeAppConfig(): void {
  if (initialized) {
    console.warn("App config already initialized");
    return;
  }

  config.value = {
    appName: window.APP_NAME || "MediaSink",
    baseUrl: window.APP_BASE || "",
    socketUrl: window.APP_SOCKETURL || "",
    apiVersion: window.APP_API_VERSION || "v1",
  };

  initialized = true;
}

/**
 * Composable to access app configuration
 * Returns readonly configuration to prevent accidental modification
 */
export function useAppConfig() {
  if (!initialized) {
    console.warn("App config not initialized, using default values");
    initializeAppConfig();
  }

  return {
    config: readonly(config),
    appName: config.value.appName,
    baseUrl: config.value.baseUrl,
    socketUrl: config.value.socketUrl,
    apiVersion: config.value.apiVersion,
  };
}
