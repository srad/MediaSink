import { type DatabaseRecording as RecordingResponse, HttpClient, MediaSinkClient } from "./MediaSinkClient";
import { useAuthStore } from "../../../stores/auth";
import { handlePotentialServerError } from "../../../utils/serverError";

const checkResponseStatus = (response: Response) => {
  if ([401].includes(response.status) && !["/login", "/register"].includes(window.location.pathname)) {
    const authStore = useAuthStore();
    authStore.logout();
    // Unauthorized: Redirect to login page
    window.location.assign("/login");
    throw new Error("Unauthorized access, redirecting to login...");
  }
};

export class MyClient extends MediaSinkClient<unknown> {
  constructor(token: string | null | undefined, apiUrl: string) {
    let auth = {};
    if (token) {
      auth = { Authorization: `Bearer ${token}` };
    }

    const client = new HttpClient({
      baseUrl: apiUrl,
      baseApiParams: {
        headers: {
          ...auth,
          "X-API-Version": window.APP_API_VERSION,
        },
      },
      /**
       * Custom fetch with error handling for server unreachability
       * @param input
       * @param init
       */
      customFetch: (input, init) => {
        return new Promise<Response>((resolve, reject) => {
          fetch(input, init)
            .then((response) => {
              checkResponseStatus(response);
              resolve(response);
            })
            .catch(async (err) => {
              // Check if error is due to server being unreachable
              const wasServerError = await handlePotentialServerError(err);
              if (wasServerError) {
                // Error was handled (user logged out and redirected)
                reject(err);
                return;
              }

              // For other errors, still try to check status
              try {
                checkResponseStatus(err);
              } catch {
                // checkResponseStatus may throw, continue
              }
              reject(err);
            });
        });
      },
    });
    super(client);
  }

  /**
   * Custom function to upload and cancel large files with progress indicator.
   * Uses XMLHttpRequest for proper upload progress tracking.
   * @param channelId Upload to which channel
   * @param file File object to upload
   * @param progress Returns the progress as number in range [0.0 ... 1.0]
   */
  channelUpload(channelId: number, file: File, progress: (pcent: number) => void): [Promise<RecordingResponse>, AbortController] {
    const controller = new AbortController();
    const authStore = useAuthStore();
    const token = authStore.getToken;
    const apiUrl = window.APP_APIURL;

    const uploadPromise = new Promise<RecordingResponse>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const formData = new FormData();
      formData.append("file", file);

      // Track upload progress
      xhr.upload.addEventListener("progress", (event) => {
        if (event.lengthComputable) {
          const progressPercent = event.loaded / event.total;
          progress(progressPercent);
        }
      });

      // Handle completion
      xhr.addEventListener("load", () => {
        try {
          if (xhr.status === 401) {
            authStore.logout();
            window.location.assign("/login");
            reject(new Error("Unauthorized access, redirecting to login..."));
            return;
          }

          if (xhr.status >= 400) {
            reject(new Error(`Upload failed with status ${xhr.status}: ${xhr.statusText}`));
            return;
          }

          const response = JSON.parse(xhr.responseText) as RecordingResponse;
          resolve(response);
        } catch (error) {
          reject(error);
        }
      });

      // Handle errors
      xhr.addEventListener("error", () => {
        reject(new Error("Upload failed: Network error"));
      });

      xhr.addEventListener("abort", () => {
        reject(new Error("Upload cancelled"));
      });

      // Setup request
      xhr.open("POST", `${apiUrl}/channels/${channelId}/upload`);

      // Add authorization header if token exists
      if (token) {
        xhr.setRequestHeader("Authorization", `Bearer ${token}`);
      }

      // Add API version header
      xhr.setRequestHeader("X-API-Version", window.APP_API_VERSION);

      // Send the request
      xhr.send(formData);

      // Link abort controller to xhr
      const originalAbort = controller.abort.bind(controller);
      controller.abort = function () {
        xhr.abort();
        originalAbort();
      };
    });

    return [uploadPromise, controller];
  }

  async isRecording(): Promise<boolean> {
    const { isRecording } = await this.recorder.recorderList();
    return isRecording;
  }

}

export const createClient = (token?: string | null | undefined, apiUrl?: string): MyClient => {
  const authStore = useAuthStore();
  return new MyClient(token || authStore.getToken, apiUrl || window.APP_APIURL);
};
