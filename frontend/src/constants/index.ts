/**
 * Application-wide constants
 */

// Video Player Constants
export const VIDEO_CONSTANTS = {
  /** Default skip duration in seconds when using skip forward/backward */
  SKIP_SECONDS: 30,
  /** Default playback speed */
  DEFAULT_PLAYBACK_SPEED: 1.0,
  /** Time update throttle in milliseconds */
  TIME_UPDATE_THROTTLE_MS: 100,
} as const;

// DataTable Constants
export const TABLE_CONSTANTS = {
  /** Default page size for tables */
  DEFAULT_PAGE_SIZE: 10,
  /** Available page size options */
  PAGE_SIZE_OPTIONS: [10, 20, 50, 100],
  /** Maximum number of visible pagination buttons */
  MAX_VISIBLE_PAGE_BUTTONS: 7,
} as const;

// WebSocket Constants
export const WEBSOCKET_CONSTANTS = {
  /** Heartbeat ping interval in milliseconds (30 seconds) */
  PING_INTERVAL_MS: 30000,
  /** Pong response timeout in milliseconds (5 seconds) */
  PONG_TIMEOUT_MS: 5000,
  /** Maximum reconnection attempts */
  MAX_RECONNECT_ATTEMPTS: 5,
  /** Base delay for exponential backoff in milliseconds (1 second) */
  RECONNECT_BASE_DELAY_MS: 1000,
} as const;

// Polling Constants
export const POLLING_CONSTANTS = {
  /** Default polling interval in milliseconds (10 seconds) */
  DEFAULT_INTERVAL_MS: 10000,
} as const;

// Settings Constants
export const SETTINGS_CONSTANTS = {
  /** Default video volume (0.0 to 1.0) */
  DEFAULT_VIDEO_VOLUME: 1.0,
  /** Default video muted state */
  DEFAULT_VIDEO_MUTED: true,
  /** Default filter view page size */
  DEFAULT_FILTER_PAGE_SIZE: 100,
} as const;

// Validation Constants
export const VALIDATION_CONSTANTS = {
  /** Maximum length for display names */
  MAX_DISPLAY_NAME_LENGTH: 100,
  /** Maximum length for tags */
  MAX_TAG_LENGTH: 50,
  /** Minimum duration for recordings in seconds */
  MIN_RECORDING_DURATION: 0,
} as const;

// UI Constants
export const UI_CONSTANTS = {
  /** Empty state container height */
  EMPTY_STATE_HEIGHT: "80vh",
  /** Toast notification duration in milliseconds */
  TOAST_DURATION_MS: 5000,
} as const;
