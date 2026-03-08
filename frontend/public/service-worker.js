/**
 * Service Worker for MediaSink PWA
 *
 * This service worker is registered by Vite PWA plugin and works in conjunction
 * with Workbox for caching and offline support.
 *
 * Update Strategy: skipWaiting is enabled in vite.config.ts, so new service
 * workers activate immediately upon installation without waiting for old
 * service workers to become unused. This triggers the controllerchange event
 * on the page, allowing us to prompt the user to reload the app.
 */

// Listen for messages from the app
self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "SKIP_WAITING") {
    // User clicked "Reload" button, activate immediately
    self.skipWaiting();
  }
});

// Workbox will handle install, activate, and fetch events automatically
// We only need to handle the user-initiated skip waiting above
