<template>
  <div @drop.prevent="handleDrop" @dragover.prevent="isDraggingOver = true" @dragleave.prevent="isDraggingOver = false" @dragend.prevent="isDraggingOver = false" :class="{ 'drag-overlay-active': isDraggingOver }" class="channel-view-container">
    <!-- Drag and drop overlay -->
    <div v-if="isDraggingOver" class="drag-overlay">
      <div class="drag-overlay-content">
        <i class="bi bi-cloud-arrow-up"></i>
        <p>Drop video file to upload</p>
      </div>
    </div>

    <BusyOverlay :visible="busyOverlay" />

    <ChannelModal @save="saveEditChannel" @close="showEditModal = false" title="Edit Channel" :saving="editFormSaving" :is-paused="editFormIsPaused" :channel-disabled="true" :clear="false" :channel-id="editFormChannelId" :show="showEditModal" :channel-name="editFormChannelName" :display-name="editFormDisplayName" :url="editFormUrl" :min-duration="editFormMinDuration" :skip-start="editFormSkipStart" />

    <JsConfirmDialog :show="showConfirm" @cancel="showConfirm = false" @confirm="deleteChannel" text="Are you sure you want to delete this channel?" />

    <ModalConfirmDialog :show="showDeleteSelectedRecordings" @cancel="showDeleteSelectedRecordings = false" @confirm="destroySelection">
      <template v-slot:header>Confirm selection</template>
      <template v-slot:body>
        <ul class="list-unstyled">
          <li :key="recording.recordingId" v-for="recording in selectedRecordings" class="list-group-item d-flex justify-content-between mb-2">
            <img class="img-thumbnail w-20 me-2" :alt="recording.filename" loading="lazy" :src="fileUrl + videoCover(recording)" />
            <div class="w-80">
              <div>{{ recording.filename }}</div>
              <div>{{ (recording.duration / 60).toFixed(1) }}min - {{ Math.fround(recording.size / 1024 / 1024 / 1024).toFixed(1) }}GB</div>
            </div>
          </li>
        </ul>
      </template>
    </ModalConfirmDialog>

    <LoadIndicator :busy="!channel">
      <ModalConfirmDialog :show="showMergeModal" @confirm="merge" @cancel="showMergeModal = false">
        <template #header>Merge videos</template>
        <template #body>
          <ul class="list-unstyled">
            <li :key="recording.recordingId" v-for="recording in selectedRecordings" class="list-group-item d-flex justify-content-between mb-2">
              <img class="img-thumbnail w-20 me-2" :alt="recording.filename" loading="lazy" :src="`${fileUrl}/${recording.videoPreview?.previewPath || recording.channelName + '/.previews/live.jpg'}`" />
              <div class="w-80">
                <div>{{ recording.filename }}</div>
                <div>{{ (recording.duration / 60).toFixed(1) }}min - {{ Math.fround(recording.size / 1024 / 1024 / 1024).toFixed(1) }}GB</div>
              </div>
            </li>
          </ul>
        </template>
      </ModalConfirmDialog>

      <ModalWindow :show="showVideoUploadModal">
        <template #header>Video Upload</template>
        <template #body>
          <div class="progress">
            <div class="progress-bar progress-bar-animated progress-bar-striped bg-info" role="progressbar" :style="{ width: `${uploadProgress * 100}%` }" aria-valuemax="1" aria-valuemin="0" aria-valuenow="0.4"></div>
          </div>
          <span class="text-center fs-6">{{ (uploadProgress * 100).toFixed(0) }}%</span>
        </template>
        <template #footer>
          <button type="button" class="btn btn-danger" @click="cancelUpload">Cancel Upload</button>
        </template>
      </ModalWindow>

      <div class="d-flex align-items-center mb-3 justify-content-between border-bottom pb-2">
        <div class="text-info fs-5 fw-bolder d-none d-sm-inline">{{ channel!.displayName }}</div>

        <div class="d-flex gap-2 align-items-center">
          <OptionsMenu v-if="!areItemsSelected" :channel-paused="channel!.isPaused" :multi-select="selectedRecordings.length === 0" @file="fileSelected" @pause="pauseChannel" @edit="editChannel" @delete="showConfirm = true" />

          <div class="d-flex gap-2 align-items-center">
            <button v-if="areItemsSelected" type="button" class="btn btn-sm btn-danger d-flex gap-2 justify-content-between" @click="showDeleteSelectedRecordings = true">
              <span>Delete selection</span>
              <i class="bi bi-trash3-fill" />
            </button>
            <button v-if="selectedRecordings.length > 1" type="button" class="btn d-flex gap-2 btn-sm btn-info justify-content-between" @click="mergeVideos">
              <span>Merge</span>
              <i class="bi bi-sign-merge-left" />
            </button>
            <button v-if="areItemsSelected" type="button" class="btn gap-2 d-flex btn-sm btn-info justify-content-between" @click="cancelSelection">
              <span>Cancel</span>
              <i class="bi bi-stop-fill" />
            </button>
            <FavButton v-if="!areItemsSelected" :faved="channel!.fav" :data="channel" @fav="bookmark" @unfav="bookmark" />
          </div>
        </div>

        <div class="d-flex gap-2 align-items-center fs-5 d-none d-sm-inline">
          <button type="button" class="btn btn-sm me-1 btn-secondary" disabled>
            <span>Count:</span>
            <span>{{ channel!.recordingsCount }}</span>
          </button>
          <button type="button" class="btn btn-sm btn-secondary" disabled>Size: {{ (channel!.recordingsSize! / 1024 / 1024 / 1024).toFixed(1) }}GB</button>
        </div>
      </div>

      <div class="row mb-5">
        <div v-for="recording in recordings" :key="recording.recordingId" class="mb-3 col-lg-6 col-xl-6 col-xxl-4 col-md-8 col-sm-8">
          <VideoItem
            @destroyed="destroyRecording"
            @checked="selectRecording"
            :job="jobStore.isProcessing(recording.recordingId)"
            :check="selectedRecordings.some((x: RecordingResponse) => x.recordingId === recording.recordingId)"
            :show-selection="true"
            :recording="recording"
            :show-title="false" />
        </div>
      </div>
    </LoadIndicator>
  </div>
</template>

<script setup lang="ts">
import VideoItem from "../components/VideoItem.vue";
import type { DatabaseRecording, DatabaseRecording as RecordingResponse, ServicesChannelInfo } from "../services/api/v1/MediaSinkClient";
import { computed, inject, onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { MessageType, SocketManager } from "../utils/socket";
import BusyOverlay from "../components/BusyOverlay.vue";
import { useToastStore } from "../stores/toast";
import { useChannelStore } from "../stores/channel";
import { useJobStore } from "../stores/job";
import OptionsMenu from "@/components/controls/OptionsMenu.vue";
import ModalConfirmDialog from "@/components/modals/ModalConfirmDialog.vue";
import ChannelModal from "@/components/modals/ChannelModal.vue";
import { createClient } from "@/services/api/v1/ClientFactory";
import LoadIndicator from "@/components/LoadIndicator.vue";
import JsConfirmDialog from "@/components/modals/JsConfirmDialog.vue";
import FavButton from "@/components/controls/FavButton.vue";
import type { ChannelUpdate } from "@/types/channel";
import ModalWindow from "@/components/modals/ModalWindow.vue";
import { videoCover } from "@/utils/video.ts";

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const route = useRoute();
const router = useRouter();

const jobStore = useJobStore();
const toastStore = useToastStore();
const channelStore = useChannelStore();

const channelId = +route.params.id as number;

const channel = ref<ServicesChannelInfo | null>(null);
const selectedRecordings = ref<RecordingResponse[]>([]);
const uploadProgress = ref(0);
const busyOverlay = ref(false);
const isDraggingOver = ref(false);

let uploadAbortController: AbortController | null = null;

// Edit channel modal state
const showEditModal = ref(false);
const editFormChannelId = ref<number>();
const editFormChannelName = ref("");
const editFormDisplayName = ref("");
const editFormIsPaused = ref(false);
const editFormUrl = ref("");
const editFormMinDuration = ref(20);
const editFormSkipStart = ref(0);
const editFormSaving = ref(false);

const showMergeModal = ref(false);
const showVideoUploadModal = ref(false);
const showConfirm = ref(false);
const showDeleteSelectedRecordings = ref(false);

const fileUrl = inject("fileUrl");

const socketManager = new SocketManager();

// --------------------------------------------------------------------------------------
// Computes
// --------------------------------------------------------------------------------------

const areItemsSelected = computed(() => selectedRecordings.value.length > 0);

const recordings = computed(() => {
  if (channel.value && channel.value.recordings) {
    return [...channel.value.recordings] // Avoid mutating an original array
      .sort((a, b) => b.videoType.localeCompare(a.videoType))
      .sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt));
  }
  return [];
});

// --------------------------------------------------------------------------------------
// Methods
// --------------------------------------------------------------------------------------

const pauseChannel = (element: HTMLInputElement): void => {
  const client = createClient();
  const fn = element.checked ? client.channels.resumeCreate : client.channels.pauseCreate;
  fn({ id: channel.value!.channelId })
    .then(() => {
      if (element.checked) {
        channelStore.resume(channel.value!.channelId);
      } else {
        channelStore.pause(channel.value!.channelId);
      }
      toastStore.info({
        title: element.checked ? "Channel resume" : "Channel pause",
        message: `Channel ${channel.value?.displayName}`,
      });
    })
    .catch((res) => alert((<{ error: string }>res).error));
};

const cancelSelection = () => (selectedRecordings.value = []);

const destroySelection = async () => {
  try {
    const client = createClient();

    for (let i = 0; i < selectedRecordings.value.length; i++) {
      const rec = selectedRecordings.value[i] as RecordingResponse;
      await client.videos.videosDelete({ id: rec.recordingId });
      const index = channel.value?.recordings?.findIndex((x: DatabaseRecording) => x.recordingId === rec.recordingId);
      if (index && index !== -1) {
        channel.value?.recordings?.splice(index, 1);
      }
    }
    // Clear selection.
    toastStore.success({
      title: "Deleted recordings",
      message: `Deleted ${selectedRecordings.value.length} files.`,
    });
    selectedRecordings.value = [];
  } catch (e) {
    toastStore.error({ title: "Deletion failed", message: (<{ message: string }>e).message });
  } finally {
    showDeleteSelectedRecordings.value = false;
  }
};

const selectRecording = ({ checked, recording }: { checked: boolean; recording: RecordingResponse }) => {
  if (checked && !selectedRecordings.value.some((x) => x.recordingId === recording.recordingId)) {
    selectedRecordings.value.push(recording);
  } else {
    selectedRecordings.value = selectedRecordings.value.filter((x) => x.recordingId !== recording.recordingId);
  }
};

const deleteChannel = async () => {
  try {
    busyOverlay.value = true;
    const client = createClient();
    await client.channels.channelsDelete({ id: channelId });
    channelStore.destroy(channelId);
    toastStore.success({
      title: "Channel deleted",
      message: `Channel ${channel.value?.displayName}`,
    });
    await router.replace("/");
  } catch (e) {
    alert(e);
  } finally {
    busyOverlay.value = false;
  }
};

const cancelUpload = () => {
  if (uploadAbortController) {
    uploadAbortController.abort();
  }
  showVideoUploadModal.value = false;
};

const fileSelected = async (file: File) => {
  try {
    console.log("[Upload] Starting file upload:", file.name);
    uploadProgress.value = 0;
    showVideoUploadModal.value = true;
    const client = createClient();
    const [req, abortController] = client.channelUpload(channelId, file, (pcent: number) => {
      console.log("[Upload] Progress:", Math.round(pcent * 100) + "%");
      uploadProgress.value = pcent;
    });
    uploadAbortController = abortController;
    console.log("[Upload] Waiting for upload to complete...");
    const recording = await req;
    console.log("[Upload] Upload completed successfully");
    uploadProgress.value = 1;
    channel.value?.recordings?.unshift(recording);
    uploadAbortController = null;
    showVideoUploadModal.value = false;
    toastStore.success({
      title: "Video uploaded",
      message: `File ${file.name} uploaded successfully`,
    });
  } catch (e) {
    console.error("[Upload] Error:", e);
    const errorMessage = e instanceof Error ? e.message : String(e);
    toastStore.error({
      title: "Upload failed",
      message: errorMessage,
    });
    showVideoUploadModal.value = false;
  }
};

const destroyRecording = (recording: RecordingResponse) => {
  if (channel.value?.recordings) {
    for (let i = 0; i < channel.value.recordings.length; i += 1) {
      if (channel.value.recordings && channel.value?.recordings[i]!.recordingId === recording.recordingId) {
        jobStore.destroy(recording.recordingId);
        channel.value?.recordings?.splice(i, 1);
        break;
      }
    }
  }
};

const bookmark = () => {
  const client = createClient();
  const fn = channel.value!.fav ? client.channels.unfavPartialUpdate : client.channels.favPartialUpdate;

  fn({ id: channel.value!.channelId })
    .then(() => (channel.value!.fav = !channel.value!.fav))
    .catch((err) => alert(err));
};

const mergeVideos = async () => {
  showMergeModal.value = true;
};

const merge = async () => {
  try {
    const client = createClient();
    await client.channels.mergeCreate({ id: channel.value!.channelId }, { recordingIds: selectedRecordings.value.map((x) => x.recordingId), reEncode: true });
  } catch (e) {
    alert(e);
  } finally {
    showMergeModal.value = false;
  }
};

const editChannel = () => {
  if (channel.value) {
    editFormChannelId.value = channel.value.channelId;
    editFormChannelName.value = channel.value.channelName;
    editFormDisplayName.value = channel.value.displayName;
    editFormIsPaused.value = channel.value.isPaused;
    editFormUrl.value = channel.value.url;
    editFormSkipStart.value = channel.value.skipStart;
    editFormMinDuration.value = channel.value.minDuration;
    showEditModal.value = true;
  }
};

const saveEditChannel = async (formData: ChannelUpdate) => {
  try {
    editFormSaving.value = true;
    await channelStore.save(formData.channelId, formData);
    showEditModal.value = false;
    // Update the channel data with the new values
    if (channel.value) {
      channel.value.displayName = formData.displayName;
      channel.value.url = formData.url;
      channel.value.minDuration = formData.minDuration;
      channel.value.skipStart = formData.skipStart;
      channel.value.isPaused = formData.isPaused;
      channel.value.channelName = formData.channelName;
    }
    toastStore.success({
      title: "Channel updated",
      message: `Channel ${formData.displayName} has been saved`,
    });
  } catch (e) {
    toastStore.error({
      title: "Error",
      message: `Failed to save channel: ${e instanceof Error ? e.message : String(e)}`,
    });
  } finally {
    editFormSaving.value = false;
  }
};

const handleDrop = (event: DragEvent) => {
  isDraggingOver.value = false;

  // Get files from drag and drop
  const files = event.dataTransfer?.files;
  if (!files || files.length === 0) {
    return;
  }

  // Filter for video files
  const videoFile = Array.from(files).find((file) => {
    return file.type.startsWith("video/");
  });

  if (!videoFile) {
    toastStore.info({
      title: "Invalid file",
      message: "Please drop a video file",
    });
    return;
  }

  // Upload the video file
  fileSelected(videoFile);
};

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onUnmounted(() => {
  channel.value = null; // Force release memory
  selectedRecordings.value = [];
  socketManager.close();
});

onMounted(async () => {
  try {
    const client = createClient();
    const data = await client.channels.channelsDetail({ id: channelId });

    if (!data) {
      await router.replace("/streams/live");
      return;
    }

    channel.value = data;

    await socketManager.connect();

    socketManager.on(MessageType.RecordingAdd, (recording) => {
      const r = recording as RecordingResponse;
      channelStore.addRecording(r);
    });

    window.scrollTo(0, 0);
  } catch (error) {
    alert((<{ error: string }>error).error);
    router.back();
  }
});
</script>

<style scoped lang="scss">
.channel-view-container {
  position: relative;
  min-height: 100vh;
}

.drag-overlay-active {
  background-color: rgba(91, 124, 250, 0.05);
}

.drag-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(91, 124, 250, 0.1);
  border: 3px dashed #5b7cfa;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  pointer-events: none;

  .drag-overlay-content {
    text-align: center;
    font-size: 2rem;
    color: #5b7cfa;
    font-weight: 600;

    i {
      display: block;
      font-size: 4rem;
      margin-bottom: 1rem;
    }

    p {
      margin: 0;
      font-size: 1.25rem;
    }
  }
}
</style>
