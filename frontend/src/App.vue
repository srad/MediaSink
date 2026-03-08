<template>
  <component :is="layoutComponent">
    <RouterView :key="route.fullPath.split('?')[0]" />
  </component>
  <div v-if="showUpdateBanner" class="update-notification">
    <p>A new version of the app is available!</p>
    <button @click="reloadApp">Reload</button>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import DefaultLayout from "./layouts/DefaultLayout.vue";
import AuthLayout from "./layouts/AuthLayout.vue";
import { useRoute } from "vue-router";
import FullscreenLayout from "./layouts/FullscreenLayout.vue";
import { useSwUpdateCheck } from "@/composables/useSwUpdateCheck";

const route = useRoute();

const layouts = {
  default: DefaultLayout,
  auth: AuthLayout,
  fullscreen: FullscreenLayout,
};

//@ts-expect-error nonsense
const layoutComponent = computed(() => layouts[route.meta.layout] || DefaultLayout);

const showUpdateBanner = ref(false);

// Periodic update checking: checks every 60 seconds (configurable)
// This ensures users are notified quickly when a new version is deployed
useSwUpdateCheck(60000); // 60000ms = 1 minute

onMounted(() => {
  if ("serviceWorker" in navigator) {
    // Listen for when a new service worker takes control of the page
    // This happens when:
    // 1. A new SW is installed (detected by periodic checks or browser default)
    // 2. skipWaiting is enabled (vite.config.ts: skipWaiting: true)
    // 3. The new SW immediately activates without waiting for old SW to become unused
    // 4. Workbox's clientsClaim() ensures the new SW controls all clients immediately
    navigator.serviceWorker.addEventListener("controllerchange", () => {
      // New service worker is now active - prompt user to reload
      showUpdateBanner.value = true;
    });
  }
});

const reloadApp = () => {
  // Signal the service worker to ensure it's ready
  if (navigator.serviceWorker.controller) {
    navigator.serviceWorker.controller.postMessage({ type: "SKIP_WAITING" });
  }

  // Reload the page to apply the new service worker and app version
  window.location.reload();
};
</script>

<style lang="scss">
@import "./assets/main.scss";

.update-notification {
  position: fixed;
  bottom: 20px;
  left: 50%;
  background-color: #333;
  color: white;
  border: 2px solid #ff5733;
  padding: 15px;
  border-radius: 5px;
  z-index: 1000;
  transform: translateX(-50%);
  min-width: 300px;
}

.update-notification button {
  margin-top: 10px;
  padding: 5px 10px;
  background-color: #ff5733;
  border: none;
  color: white;
  cursor: pointer;
  border-radius: 3px;
}

.update-notification button:hover {
  background-color: #ff2e00;
}
</style>
