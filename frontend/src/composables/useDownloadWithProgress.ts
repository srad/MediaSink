import { onUnmounted, readonly, ref, watch } from "vue";

// Helper function to trigger the browser's download dialog
function triggerDownload(blob: Blob, filename: string) {
  const objectUrl = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = objectUrl;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(objectUrl);
}

export function useDownloadWithProgress() {
  const isDownloading = ref(false);
  const downloadProgress = ref<number | null>(null);
  const error = ref<string | null>(null);
  let abortController: AbortController | null = null;

  const handleBeforeUnload = (event: BeforeUnloadEvent) => {
    event.preventDefault();
    event.returnValue = "A download is active. Are you sure you want to leave the page? Files may be lost.";
    return event.returnValue; // For older browsers
  };

  // Watch isDownloading to add/remove the listener
  watch(isDownloading, (newValue) => {
    if (newValue) {
      window.addEventListener("beforeunload", handleBeforeUnload);
    } else {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    }
  });

  // Ensure the listener is removed when the composable is "destroyed"
  // (e.g., when the component using it is unmounted)
  onUnmounted(() => {
    if (isDownloading.value) {
      // Only remove if it was still active
      window.removeEventListener("beforeunload", handleBeforeUnload);
    }
  });

  async function startDownload(url: string, token?: string, clientVersion?: string, filename: string = "") {
    if (isDownloading.value) {
      console.warn("A download is already in progress.");
      return; // Prevent starting multiple downloads simultaneously via the same composable instance
    }
    isDownloading.value = true;
    downloadProgress.value = 0;
    error.value = null;
    abortController = new AbortController();

    try {
      const headers = new Headers();
      if (token) {
        headers.append("Authorization", `Bearer ${token}`);
      }
      if (clientVersion) {
        headers.append("X-API-Version", clientVersion);
      }

      const response = await fetch(url, {
        method: "GET",
        headers: headers,
        signal: abortController.signal, // Pass the signal to fetch
      });

      if (!response.ok) {
        throw new Error(`Network response was not OK: ${response.status} ${response.statusText}`);
      }

      const contentLength = response.headers.get("Content-Length");
      if (!contentLength) {
        console.warn("Content-Length header missing. Progress indicator not possible. Fallback to direct blob download.");
        // Fallback to direct blob download without progress
        const blob = await response.blob();
        const effectiveFilename = filename || url.substring(url.lastIndexOf("/") + 1) || "downloaded-file";
        triggerDownload(blob, effectiveFilename);
        return; // Early exit as progress cannot be tracked
      }

      const totalLength = parseInt(contentLength, 10);
      let receivedLength = 0;
      const chunks: Uint8Array[] = [];

      // Ensure response.body exists (TypeScript non-null assertion operator !)
      // This is safe as response.ok was checked and a body is expected.
      const reader = response.body!.getReader();

      while (true) {
        // Check if the download was aborted before reading further
        if (abortController.signal.aborted) {
          await reader.cancel("Download canceled by user."); // Gracefully cancel the stream
          console.log("Download canceled by user.");
          error.value = "Download canceled by user.";
          break; // Exit the loop, do not throw an error for intentional cancellation
        }

        const { done, value } = await reader.read();
        if (done) {
          break;
        }

        chunks.push(value); // value is Uint8Array
        receivedLength += value.length;
        downloadProgress.value = Math.round((receivedLength / totalLength) * 100);
      }

      // Only trigger download if not aborted
      if (!abortController.signal.aborted) {
        const blob = new Blob(chunks as any[]);
        const effectiveFilename = filename || url.substring(url.lastIndexOf("/") + 1) || "downloaded-file";
        triggerDownload(blob, effectiveFilename);
      }
    } catch (e: any) {
      // Catch actual errors, including AbortError if fetch itself throws it (e.g., aborted before the response body is accessed)
      if (e.name === "AbortError") {
        // This case handles AbortError thrown by fetch itself, not our manual break from the loop.
        console.log("Download was aborted (likely by fetch).");
        if (!error.value) {
          // Set error only if not already set by our manual cancellation
          error.value = ""; // 'Download aborted.';
        }
      } else {
        console.error("Download failed:", e);
        error.value = e.message || "File could not be downloaded.";
      }
    } finally {
      isDownloading.value = false; // This will trigger the watcher to remove the beforeunload listener
      // downloadProgress.value = null; // Optional: reset progress, or let it stay at the last value.
      abortController = null; // Reset the controller
    }
  }

  function cancelDownload() {
    if (abortController) {
      abortController.abort(); // This will set abortController.signal.aborted to true
      console.log("Download cancellation requested.");
    }
  }

  return {
    isDownloading: readonly(isDownloading),
    downloadProgress: readonly(downloadProgress),
    error: readonly(error),
    startDownload,
    cancelDownload,
  };
}
