import { ref, onUnmounted } from "vue";
import { MessageType, SocketManager } from "../utils/socket";

const isConnected = ref(false);

export function useSocketStatus() {
  const manager = new SocketManager();

  manager
    .connect()
    .then(() => {
      isConnected.value = true;
    })
    .catch(() => {
      isConnected.value = false;
    });

  const handleClose = () => {
    isConnected.value = false;
  };

  const handlePong = () => {
    isConnected.value = true; // Optional: based on heartbeat messages
  };

  manager.on(MessageType.Pong, handlePong);

  // Could also listen to connection events, if you fire them explicitly
  window.addEventListener("beforeunload", () => manager.close());

  onUnmounted(() => {
    manager.off(MessageType.Pong, handlePong);
    manager.close();
  });

  return {
    isConnected,
    socketManager: manager,
  };
}
