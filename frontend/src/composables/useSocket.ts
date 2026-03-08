import { onUnmounted } from "vue";
import { SocketManager, type MessageTypeKey } from "../utils/socket";

// Singleton socket manager instance
let socketManagerInstance: SocketManager | null = null;

export function useSocket() {
  // Create singleton instance if it doesn't exist
  if (!socketManagerInstance) {
    socketManagerInstance = new SocketManager();
  }

  const socket = socketManagerInstance;

  // Clean up instance listeners when component unmounts
  onUnmounted(() => {
    // Note: We don't close the connection here as it's shared across components
    // Connection cleanup should be handled at app level (e.g., on logout)
  });

  return {
    socket,
    connect: () => socket.connect(),
    on: <T = unknown>(event: MessageTypeKey, fn: (data: T) => void) => socket.on(event, fn),
    off: <T = unknown>(event: MessageTypeKey, fn: (data: T) => void) => socket.off(event, fn),
    close: () => socket.close(),
  };
}
