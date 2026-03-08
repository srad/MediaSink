import { useAuthStore } from "../stores/auth";
import { handleServerUnreachable } from "./serverError";

declare global {
  interface Window {
    APP_SOCKETURL: string;
    APP_API_VERSION: string;
  }
}

type ListenerType<T = unknown> = (data: T) => void;

interface SocketPayload<T = unknown> {
  name: MessageTypeKey;
  data: T;
}

export class SocketManager {
  private static connection: WebSocket | null = null;

  private static listeners: Record<MessageTypeKey, ListenerType[]> = {
    "channel:offline": [],
    "channel:online": [],
    "channel:start": [],
    "channel:thumbnail": [],
    "job:activate": [],
    "job:create": [],
    "job:deactivate": [],
    "job:delete": [],
    "job:deleted": [],
    "job:done": [],
    "job:preview:done": [],
    "job:preview:progress": [],
    "job:progress": [],
    "job:start": [],
    "recording:add": [],
    pong: [],
  };

  private instanceListeners: Record<MessageTypeKey, ListenerType[]> = {
    "channel:offline": [],
    "channel:online": [],
    "channel:start": [],
    "channel:thumbnail": [],
    "job:activate": [],
    "job:create": [],
    "job:deactivate": [],
    "job:delete": [],
    "job:deleted": [],
    "job:done": [],
    "job:preview:done": [],
    "job:preview:progress": [],
    "job:progress": [],
    "job:start": [],
    "recording:add": [],
    pong: [],
  };
  private static pendingPromise: Promise<WebSocket> | null = null;
  private static reconnectAttempts = 0;
  private static maxReconnectAttempts = 5;
  private static pingInterval: ReturnType<typeof setInterval> | null = null;
  private static reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private static intentionalClose = false;
  private static pongTimeout: ReturnType<typeof setTimeout> | null = null;

  connect(): Promise<WebSocket> {
    // If a connection is already open, return it.
    if (SocketManager.connection !== null && SocketManager.connection.readyState === WebSocket.OPEN) {
      return Promise.resolve(SocketManager.connection);
    }

    // If there's an ongoing connection attempt, return its promise.
    if (SocketManager.pendingPromise) {
      return SocketManager.pendingPromise;
    }

    SocketManager.pendingPromise = new Promise<WebSocket>((resolve, reject) => {
      const authStore = useAuthStore();

      if (!authStore.isLoggedIn) {
        SocketManager.pendingPromise = null; // Clear pending promise
        reject("WebSocket missing authorization token");
        return;
      }

      // Create the WebSocket connection
      SocketManager.connection = new WebSocket(window.APP_SOCKETURL + "?Authorization=" + authStore.getToken + `&ApiVersion=${window.APP_API_VERSION}`);

      SocketManager.connection.addEventListener("message", (message) => SocketManager.handleMessage(message));

      SocketManager.connection.addEventListener("open", () => {
        console.log("WebSocket connection opened");

        // Reset reconnection attempts on successful connection
        SocketManager.reconnectAttempts = 0;

        // Start heartbeat ping/pong
        SocketManager.startHeartbeat();

        resolve(SocketManager.connection!);
        SocketManager.pendingPromise = null; // Clear pending promise
      });

      SocketManager.connection.addEventListener("close", () => {
        console.log("WebSocket connection closed");

        // Stop heartbeat
        SocketManager.stopHeartbeat();

        if (SocketManager.pendingPromise) {
          SocketManager.pendingPromise = Promise.reject(new Error("WebSocket connection closed"));
        }
        SocketManager.pendingPromise = null;

        // Attempt reconnection if it wasn't an intentional close
        if (!SocketManager.intentionalClose) {
          SocketManager.attemptReconnect();
        } else {
          // Reset flag for future connections
          SocketManager.intentionalClose = false;
        }
      });

      SocketManager.connection.addEventListener("error", (ev: Event) => {
        console.error("WebSocket error", ev);
        SocketManager.connection?.close();
        SocketManager.connection = null;
        SocketManager.pendingPromise = null; // Clear pending promise
        reject(ev);
      });
    });

    return SocketManager.pendingPromise;
  }

  close(): void {
    this.offGlobal();

    this.instanceListeners = {
      "channel:online": [],
      "channel:start": [],
      "channel:thumbnail": [],
      "job:activate": [],
      "job:create": [],
      "job:deactivate": [],
      "job:delete": [],
      "job:deleted": [],
      "job:done": [],
      "job:preview:done": [],
      "job:preview:progress": [],
      "job:progress": [],
      "job:start": [],
      "recording:add": [],
      pong: [],
      "channel:offline": [],
    };

    // Mark as intentional close to prevent reconnection
    SocketManager.intentionalClose = true;

    // Clear any pending reconnection attempts
    if (SocketManager.reconnectTimeout !== null) {
      clearTimeout(SocketManager.reconnectTimeout);
      SocketManager.reconnectTimeout = null;
    }

    // Stop heartbeat
    SocketManager.stopHeartbeat();

    SocketManager.connection?.close();
    SocketManager.connection = null;
    SocketManager.pendingPromise = null;
  }

  on<T = unknown>(event: MessageTypeKey, fn: ListenerType<T>): void {
    if (!this.instanceListeners[event]) {
      this.instanceListeners[event] = [];
    }
    if (!SocketManager.listeners[event]) {
      SocketManager.listeners[event] = [];
    }

    SocketManager.listeners[event].push(fn as ListenerType);
    this.instanceListeners[event].push(fn as ListenerType);
  }

  off<T = unknown>(event: MessageTypeKey, fn: ListenerType<T>): void {
    this.offInstance(event, fn);
    this.offGlobalFn(event, fn);
  }

  private static handleMessage(raw: MessageEvent<string>): void {
    try {
      const parsed = JSON.parse(raw.data) as SocketPayload;
      SocketManager.emit(parsed.name, parsed.data);
    } catch (e) {
      console.error("Failed to parse WebSocket message", e);
    }
  }

  private static emit<T = unknown>(event: MessageTypeKey, data: T): void {
    const listeners = SocketManager.listeners[event];
    if (listeners) {
      listeners.forEach((fn) => fn(data));
    }
  }

  private offInstance<T>(event: MessageTypeKey, fn: ListenerType<T>): void {
    this.instanceListeners[event] = (this.instanceListeners[event] || []).filter((listener) => listener !== fn);
  }

  private offGlobalFn<T>(event: MessageTypeKey, fn: ListenerType<T>): void {
    SocketManager.listeners[event] = (SocketManager.listeners[event] || []).filter((listener) => listener !== fn);
  }

  private offGlobal(): void {
    const allInstanceFns = Object.values(this.instanceListeners).flat();
    for (const event of Object.keys(SocketManager.listeners) as MessageTypeKey[]) {
      SocketManager.listeners[event] = SocketManager.listeners[event].filter((fn) => !allInstanceFns.includes(fn));
    }
  }

  /**
   * Attempts to reconnect with exponential backoff
   */
  private static attemptReconnect(): void {
    if (SocketManager.reconnectAttempts >= SocketManager.maxReconnectAttempts) {
      console.error(`WebSocket reconnection failed after ${SocketManager.maxReconnectAttempts} attempts`);
      SocketManager.reconnectAttempts = 0;
      return;
    }

    // Calculate exponential backoff: 1s, 2s, 4s, 8s, 16s
    const delay = Math.pow(2, SocketManager.reconnectAttempts) * 1000;
    SocketManager.reconnectAttempts++;

    console.log(`WebSocket attempting to reconnect in ${delay}ms (attempt ${SocketManager.reconnectAttempts}/${SocketManager.maxReconnectAttempts})`);

    SocketManager.reconnectTimeout = setTimeout(() => {
      const manager = new SocketManager();
      manager.connect().catch((error) => {
        console.error("WebSocket reconnection attempt failed:", error);
      });
    }, delay);
  }

  /**
   * Starts the heartbeat ping/pong mechanism
   */
  private static startHeartbeat(): void {
    // Clear any existing heartbeat
    SocketManager.stopHeartbeat();

    // Send ping every 30 seconds
    SocketManager.pingInterval = setInterval(() => {
      if (SocketManager.connection?.readyState === WebSocket.OPEN) {
        SocketManager.connection.send(JSON.stringify({ name: "ping" }));

        // Set a timeout to detect if pong is not received
        SocketManager.pongTimeout = setTimeout(() => {
          console.warn("WebSocket pong not received, connection may be dead");
          // Close the connection to trigger reconnection
          SocketManager.connection?.close();
        }, 5000); // Wait 5 seconds for pong response
      }
    }, 30000);

    // Listen for pong responses
    const pongHandler = () => {
      // Clear the pong timeout when we receive a pong
      if (SocketManager.pongTimeout !== null) {
        clearTimeout(SocketManager.pongTimeout);
        SocketManager.pongTimeout = null;
      }
    };

    // Add pong listener if not already present
    if (!SocketManager.listeners["pong"].includes(pongHandler)) {
      SocketManager.listeners["pong"].push(pongHandler);
    }
  }

  /**
   * Stops the heartbeat ping/pong mechanism
   */
  private static stopHeartbeat(): void {
    if (SocketManager.pingInterval !== null) {
      clearInterval(SocketManager.pingInterval);
      SocketManager.pingInterval = null;
    }

    if (SocketManager.pongTimeout !== null) {
      clearTimeout(SocketManager.pongTimeout);
      SocketManager.pongTimeout = null;
    }
  }
}

export const MessageType = {
  ChannelOnline: "channel:online",
  ChannelOffline: "channel:offline",
  ChannelThumbnail: "channel:thumbnail",
  ChannelStart: "channel:start",
  RecordingAdd: "recording:add",
  JobActivate: "job:activate",
  JobDone: "job:done",
  JobDeactivate: "job:deactivate",
  JobStart: "job:start",
  JobCreate: "job:create",
  JobDelete: "job:delete",
  JobPreviewDone: "job:preview:done",
  JobProgress: "job:progress",
  JobDeleted: "job:deleted",
  JobPreviewProgress: "job:preview:progress",
  Pong: "pong",
} as const;

export type MessageTypeKey = (typeof MessageType)[keyof typeof MessageType];
