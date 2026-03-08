# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MediaSink.Vue is a Vue 3 TypeScript-based web application for managing media streams, recordings, and video editing. It features real-time socket communication, Pinia state management, and Bootstrap 5 UI with support for light/dark themes. The app is a Progressive Web App (PWA) with offline capabilities.

## Development Commands

```sh
# Install dependencies
pnpm install

# Development server with hot-reload
pnpm dev

# Type-check, build, and minify for production
pnpm build

# Preview production build locally
pnpm preview

# Run unit tests
pnpm test:unit

# Run single unit test file
pnpm test:unit [file-path]

# Generate coverage report
pnpm coverage

# Run end-to-end tests
pnpm test:e2e

# Run E2E tests on specific browser
pnpm test:e2e --env chrome
pnpm test:e2e --env firefox

# Run E2E tests for specific file
pnpm test:e2e tests/e2e/example.ts

# Run E2E tests in debug mode
pnpm test:e2e --debug

# Lint and auto-fix with ESLint
pnpm lint

# Format code with Prettier
pnpm format

# Generate API client from Swagger spec
pnpm client

# Approve dependency builds (pnpm 10+)
pnpm approve-builds
```

## Architecture

### Core Stack
- **Framework**: Vue 3 (Composition API with `<script setup>`)
- **State Management**: Pinia with persistedstate plugin
- **Routing**: Vue Router with auth guards
- **API Client**: Auto-generated from Swagger (src/services/api/v1/MediaSinkClient.ts)
- **Real-time**: Custom WebSocket manager (src/utils/socket.ts)
- **Styling**: Bootstrap 5 + SCSS variables
- **Internationalization**: Vue i18n
- **Testing**: Vitest (unit), Nightwatch (E2E)
- **PWA**: Vite PWA plugin with Workbox

### Directory Structure

```
src/
├── components/          # Reusable Vue components
│   ├── modals/         # Modal dialogs (ChannelModal, VideoEnhancementModal, etc.)
│   ├── channels/       # Channel-specific components
│   ├── charts/         # Chart components (CPU, Traffic)
│   ├── controls/       # Control buttons and menus
│   ├── navs/           # Navigation components
│   └── *.vue           # Top-level components
├── views/              # Page-level components (routed)
├── router/             # Vue Router configuration and routes
├── stores/             # Pinia state stores (auth, channel, job, settings, toast)
├── services/           # Business logic and API integration
│   └── api/v1/         # Auto-generated Swagger API client
├── composables/        # Vue 3 composables (socket, app config, download)
├── layouts/            # Layout wrappers (AuthLayout, DefaultLayout, FullscreenLayout)
├── utils/              # Utility functions (socket, error handling, datetime, etc.)
├── types/              # TypeScript type definitions
├── constants/          # App constants
├── i18n/               # Internationalization setup
├── locales/            # Translation files (en/*)
├── assets/             # Static assets and global styles
├── main.ts             # Vue app entry point
└── App.vue             # Root component

public/
├── windows11/          # Windows 11 PWA icons
├── android/            # Android PWA icons
└── ios/                # iOS PWA icons
```

### Key Architectural Patterns

**API Communication**
- API client auto-generated from Swagger spec: `MediaSinkClient` in `src/services/api/v1/MediaSinkClient.ts`
- Client factory: `src/services/api/v1/ClientFactory.ts` - creates authenticated clients with token and API URL
- 401 responses trigger logout and redirect to login
- Custom `channelUpload()` method with progress callback and abort controller
- App config from window globals (APP_APIURL, APP_SOCKETURL, APP_FILEURL, etc.)

**State Management (Pinia)**
- Stores use both Setup Store and Options Store patterns:
  - **Setup Store** (function-based): `auth.ts`, `settings.ts` - more flexible with TypeScript
  - **Options Store** (object-based): `channel.ts`, `job.ts`, `toast.ts` - more traditional pattern
- Stores persist state to localStorage via pinia-plugin-persistedstate
- Core stores: auth (token, login state), channel, job, settings, toast
- All state mutations must go through actions, never direct mutations from components

**Real-time Communication**
- Singleton `SocketManager` in `src/utils/socket.ts`
- Accessed via `useSocket()` composable: `src/composables/useSocket.ts`
- Listeners are not cleaned up on component unmount (shared across app)
- Connection cleanup on logout

**Components**
- Use Composition API with `<script setup lang="ts">`
- Define props, emits, and expose explicitly
- Example: `DataTable.vue` - feature-rich sortable/searchable table with localStorage persistence
- Scoped SCSS with Bootstrap 5 variables from `src/assets/custom-bootstrap.scss`

**Routing**
- Auth guard in router.beforeEach - protects routes, redirects unauthenticated users to /login
- Route meta: `layout` (auth, default, fullscreen) and `title` for page names
- Automatic scroll to top on navigation

### Important Config Files
- `vite.config.ts` - Vite build config with Vue, DevTools, Nightwatch, PWA plugins
- `eslint.config.ts` - ESLint with Vue, TypeScript, Vitest plugins; enforces double quotes
- `tsconfig.app.json`, `tsconfig.node.json`, `tsconfig.vitest.json` - TypeScript configurations
- `package.json` - Dependencies, build scripts, pnpm config for ignored/built deps

## Code Patterns & Conventions

**TypeScript & Quotes**
- ESLint enforces **double quotes** for all TypeScript and Vue files
- Always use `<script setup lang="ts">` in components
- Define prop types with `defineProps<T>()` and emits with `defineEmits<T>()`

**Component Props and Emits**
```typescript
// Good example from DataTable.vue
const props = defineProps<{
  columns: Column[];
  data: TableRow[];
  defaultSortKey?: string;
  pageSize?: number;
  storageKey?: string;
}>();

const emit = defineEmits<{
  (event: "page-change", page: number): void;
  (event: "page-size-change", size: number): void;
}>();
```

**Computed and Reactive State**
- Use `ref()` for reactive state, `computed()` for derived state
- Use `watch()` for side effects
- Prefer computed properties over watchers for transformations

**API Calls**
```typescript
import { createClient } from "@/services/api/v1/ClientFactory";
const client = createClient(); // Gets token from auth store automatically
const data = await client.someEndpoint.list();
```

**Error Handling for Server Unreachability**
- Server errors (network failures, unreachable API) are automatically detected in `src/services/api/v1/ClientFactory.ts`
- When server is unreachable:
  1. User is logged out automatically via `handleServerUnreachable()` in `src/utils/serverError.ts`
  2. Error toast notification is displayed with message "Connection Error"
  3. User is redirected to `/login` page
- Function `handlePotentialServerError()` detects network errors (fetch failures, ECONNREFUSED, etc.)
- Components can also use `isServerUnreachable()` to check if error is network-related
- 401 Unauthorized errors are handled separately (redirect to login without error toast)

**Socket Usage**
```typescript
import { useSocket } from "@/composables/useSocket";
const { on, off, connect } = useSocket();

// Register listener
on<SomeType>("EVENT_NAME", (data) => {
  // Handle event
});

// Clean up (optional, shared across app)
off<SomeType>("EVENT_NAME", handler);
```

**Pinia Store Pattern**

Setup Store syntax (function-based):
```typescript
export const useMyStore = defineStore("storeName", () => {
  const state = ref<Type>(initialValue);
  const computedValue = computed(() => /* ... */);

  const action = async () => { /* ... */ };

  return { state, computedValue, action };
}, { persist: true });
```

Options Store syntax (object-based, also used in codebase):
```typescript
export const useMyStore = defineStore("storeName", {
  state: () => ({ items: [] as Item[] }),
  getters: {
    all: (state) => state.items,
  },
  actions: {
    async load() {
      this.items = await fetchItems();
    },
  },
});
```

**Translations (i18n)**
- Translation files in `src/locales/en/`
- Use in components: `{{ t("key.path") }}`
- Define translations in locale files before using

**Form Validation**
- Use custom `FieldValidator` framework from `src/utils/validator.ts`
- Define validators with `createValidator()` and `FieldConfig[]` array
- Each field config has name, validator function, validMessage, and invalidMessage
- Call `validator.validateAll()` to get validation results with `{ message, isValid }` for each field
- Display errors with `<AppAlert>` and `<CheckList>` components (see ChannelModal.vue pattern)
- Store validation messages in `validations` ref and clear on form reset/cancel

**Styling**
- Import Bootstrap variables: `@use "@/assets/custom-bootstrap.scss" as bootstrap;`
- Use scoped SCSS in components: `<style scoped lang="scss">`
- Custom Bootstrap overrides in `src/assets/custom-bootstrap.scss`
- Light/dark theme support via `[data-bs-theme="light"]` and `[data-bs-theme="dark"]` selectors

## Git & Commits

- Main branch: `main` (use for PRs)
- Recent commits show focus on composables, dependencies, and DataTable enhancements
- Always run `pnpm lint` and `pnpm format` before committing
- Run tests (`pnpm test:unit`, `pnpm test:e2e`) to verify changes

## Key Features

### Video Analysis (Scenes & Highlights)

The app displays automated scene and motion detection analysis on videos:

**Components:**
- `src/components/VideoStripe.vue` - Timeline with scene boundaries (colored lines) and motion highlights (colored bars)
- `src/views/VideoView.vue` - Video player with analysis controls

**Analysis Controls in VideoView:**
1. **Mode Toggle** - Switch between "Highlights" and "Scenes" display
2. **Navigation** - Previous/Next buttons to jump between points
3. **Index Selector** - Dropdown to directly select any scene or highlight (shows position, timestamp, and intensity percentage)

**How it Works:**
- Scene boundaries rendered as vertical colored lines (based on `changeIntensity`: gold/orange/red)
- Motion highlights rendered as vertical colored bars (based on intensity: green/yellow/orange/red)
- Both are positioned inside the scrollable `VideoStripe` container and scale with zoom
- Default mode: "Highlights"
- Overlay positioned inside `imageRowContainer` to scroll with video frames
- All overlays have `pointer-events: none` to not interfere with user interactions

**Related API:**
- Endpoint: `GET /api/v1/analysis/{recordingId}` (auto-queried on VideoView mount)
- Response includes `scenes[]` and `highlights[]` arrays with timing and intensity data
- Status field indicates: `null` (not analyzed), `"pending"`, `"processing"`, or `"completed"`

## Key Files to Know

- `src/App.vue` - Root component with layout routing
- `src/services/api/v1/ClientFactory.ts` - API client factory with auth handling and server error detection
- `src/utils/serverError.ts` - Server unreachability utilities (detect network errors, logout user, show error toast)
- `src/utils/validator.ts` - Custom form validation framework (FieldValidator, createValidator)
- `src/stores/auth.ts` - Authentication store (Pinia)
- `src/composables/useSocket.ts` - WebSocket singleton composable
- `src/components/modals/ChannelModal.vue` - Example modal with validation pattern
- `src/components/VideoEnhancementModal.vue` - Self-contained modal component for video enhancement
- `src/components/DataTable.vue` - Example of a complex, feature-rich component
- `src/components/VideoStripe.vue` - Video timeline with frames, analysis overlays, and selection markers
- `src/layouts/DefaultLayout.vue` - Main layout with socket event listeners (register/unregister handlers on mount)
- `src/router/index.ts` - Route definitions and auth guards
- `src/utils/socket.ts` - WebSocket manager implementation
- `src/views/VideoView.vue` - Video player and editor with analysis controls
- `vite.config.ts` - Build and plugin configuration

## Testing

- Unit tests use Vitest with `@vue/test-utils`
- E2E tests use Nightwatch with Vue integration
- Mock Service Worker (MSW) for API mocking in tests
- Test files: `src/**/__tests__/*`

## Common Development Tasks

**Adding a new page**
1. Create `.vue` component in `src/views/`
2. Add route in `src/router/index.ts` with proper meta (layout, title)
3. Use `useAuthStore()` if auth-required; it's auto-redirected by router guard

**Adding a store**
1. Create new file in `src/stores/`
2. Use either Setup Store (function-based) or Options Store (object-based) syntax - both are used in the codebase
3. Add `{ persist: true }` option to persist to localStorage if needed
4. All state changes must go through actions - never mutate directly from components
5. Import `useMyStore` where needed

**Using the API client**
1. Import factory: `import { createClient } from "@/services/api/v1/ClientFactory"`
2. Create client: `const client = createClient()` (auto-authenticated)
3. Call methods: `await client.channels.list()`, `await client.recorder.recorderList()`
4. For file uploads: use `client.channelUpload(channelId, file, progressCallback)`

**Real-time events**
1. Use `useSocket()` composable
2. Register listener: `on<DataType>("EVENT_NAME", handler)`
3. Cleanup: `off<DataType>("EVENT_NAME", handler)` (optional, shared)

**Adding translations**
1. Add key-value pair in `src/locales/en/*.ts` file
2. Use in template: `{{ t("namespace.key") }}`
3. Import i18n in component if needed: `import { useI18n } from "vue-i18n"`

## Notes

- Service Worker auto-registers via virtual module
- PWA manifest configured for Windows 11, Android, and iOS icons
- App configuration injected at build time via window globals
- Port configuration: typically runs on `http://localhost:5173` in dev
- CORS and API URL configured via `APP_APIURL` window variable

### Core BASH Tools (NO EXCEPTIONS)

# Pattern Search - USE 'rg' ONLY
rg -n "pattern" --glob '!node_modules/*'
rg -l "pattern"              # List matching files
rg -t py "pattern"           # Search Python files only

# File Finding - USE 'fd' ONLY
fd filename                  # Find by name
fd -e py                     # Find Python files
fd -H .env                   # Include hidden