<template>
  <ul class="list-group list-group-flush videoinfo-list">
    <li class="list-group-item d-flex justify-content-evenly gap-1 px-1 text-muted small">
      <div class="d-flex align-items-center" style="min-width: 0; overflow: hidden; white-space: nowrap" title="Duration">
        <i class="bi bi-stopwatch-fill me-1" />
        <span class="text-truncate">{{ t("recording.durationMinutes", [durationFormatted]) }}</span>
      </div>

      <div class="d-flex align-items-center" style="min-width: 0; overflow: hidden; white-space: nowrap" title="Size">
        <i class="bi bi-sd-card-fill me-1" />
        <span class="text-truncate">{{ (size / 1024 / 1024 / 1024).toFixed(1) }}GB</span>
      </div>

      <div class="d-flex align-items-center" style="min-width: 0; overflow: hidden; white-space: nowrap" title="Upload time">
        <i class="bi bi-calendar4-event me-1" />
        <span class="text-truncate">{{ ago }}</span>
      </div>
    </li>
    <li v-if="expand" class="list-group-item d-flex justify-content-between">
      <span>{{ t("recording.bitRate") }}</span>
      <span>{{ (bitRate / 1024 / 1024).toFixed(2) }} MBit</span>
    </li>
    <li v-if="expand" class="list-group-item d-flex justify-content-between">
      <span>{{ t("recording.resolution") }}</span>
      <span>{{ width }}x{{ props.height }}</span>
    </li>
    <li v-if="expand" class="list-group-item d-flex justify-content-between">
      <span>{{ t("recording.started") }}</span>
      <span>{{ createdAtFormatted }}</span>
    </li>
    <li v-if="expand" class="list-group-item d-flex justify-content-between">
      <div>{{ t("recording.convert") }}</div>
      <div class="btn-group">
        <button v-if="props.height !== 720" class="btn btn-light" @click="emit('convert', { recording: data, mediaType: '720' })">720p</button>
        <button v-if="props.height !== 1080" class="btn btn-light" @click="emit('convert', { recording: data, mediaType: '1080' })">1080p</button>
        <!--<button class="btn btn-sm btn-warning" @click="emit('convert', {recording: data, mediaType: 'mp3'})">MP3</button>-->
      </div>
    </li>
    <li v-if="false" @click="expand = !expand" class="expand p-0 list-group-item d-flex justify-content-center expand-control" :class="{ expanded: expand }">
      <i v-if="!expand" class="bi bi-caret-down-fill"></i>
      <i v-else class="bi bi-caret-up-fill"></i>
    </li>
    <li v-if="isDownloading && downloadProgress !== null" class="list-group-item align-middle video-controls rounded-bottom-2">
      <div class="d-flex justify-between justify-content-center gap-2">
        <div class="flex-grow-1">
          <ProgressBar :percent="downloadProgress" />
        </div>
        <button type="button" class="btn btn-sm btn-danger py-0 px-1" @click="cancelDownload">Cancel</button>
      </div>
    </li>
    <li v-else class="list-group-item fs-6 video-controls rounded-bottom-2">
      <div class="justify-content-between d-flex">
        <div class="d-flex gap-2">
          <a class="btn-link btn-sm p-0 px-2" @click="download">
            <i class="bi bi-download"></i>
          </a>
          <FavButton :data="data" :faved="bookmark" @fav="emit('bookmarked', props.data, false)" @unfav="emit('bookmarked', props.data, true)" />
        </div>

        <div class="d-flex gap-3">
          <button type="button" class="btn btn-sm p-0" @click="emit('analysis', data)">
            <i class="bi bi-eye"></i>
          </button>
          <button type="button" class="btn btn-sm p-0" @click="emit('preview', data)">
            <i class="bi bi-film"></i>
          </button>
          <button type="button" class="btn btn-sm p-0" @click="emit('enhance', data)">
            <i class="bi bi-magic"></i>
          </button>
          <button type="button" class="btn btn-sm p-0" @click="emit('destroy', data)">
            <i class="bi bi-trash3-fill text-danger" />
          </button>
        </div>
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { DatabaseRecording as RecordingResponse } from "../services/api/v1/MediaSinkClient.ts";
import { fromNow } from "../utils/datetime";
import FavButton from "./controls/FavButton.vue";
import { useAuthStore } from "../stores/auth";
import { useDownloadWithProgress } from "../composables/useDownloadWithProgress";
import ProgressBar from "../components/ProgressBar.vue";

const authStore = useAuthStore();

const { t } = useI18n();

const emit = defineEmits<{
  (e: "preview", value: RecordingResponse): void;
  (e: "enhance", value: RecordingResponse): void;
  (e: "destroy", value: RecordingResponse): void;
  (e: "analysis", value: RecordingResponse): void;
  (e: "bookmarked", data: RecordingResponse, value: boolean): void;
  (e: "convert", value: { recording: RecordingResponse; mediaType: string }): void;
}>();

const props = defineProps<{
  //index: number
  url: string;
  bookmark: boolean;
  data: RecordingResponse;
  duration: number;
  size: number;
  bitRate: number;
  width: number;
  height: number;
  createdAt: string;
  disableButtons?: boolean;
}>();

const durationFormatted = computed(() => (props.duration / 60).toFixed(2));

const createdAtFormatted = computed(() =>
  new Date(props.createdAt).toLocaleDateString(undefined, {
    hour: "numeric",
    minute: "numeric",
  }),
);

const ago = ref(fromNow(Date.parse(props.createdAt)));

const expand = ref(false);

const {
  isDownloading,
  downloadProgress,
  error: downloadError,
  startDownload,
  cancelDownload, // Die neue Funktion aus dem Composable
} = useDownloadWithProgress();

const checkDownload = () => {};

onMounted(() => {
  window.addEventListener("beforeunload", checkDownload);
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", checkDownload);
});

const download = async () => {
  const authToken = authStore.getToken!;

  const filenameToUse = props.data.filename || `video-${props.data.recordingId}.mp4`;

  await startDownload(props.url, authToken, window.APP_API_VERSION, filenameToUse);

  if (downloadError.value && downloadError.value !== "Download aborted") {
    console.error("Download error:", downloadError.value);
    alert(downloadError.value);
  }
};
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;

[data-bs-theme="light"] {
  .video-controls {
    background: bootstrap.$info-bg-subtle;
  }
}

[data-bs-theme="dark"] {
  .expand-control {
    background: #4b4e6d;
    color: bootstrap.$white;
  }

  .video-controls {
    background: bootstrap.$primary;
    color: bootstrap.$white;
  }
}

.expand {
  background: ghostwhite;
}

.btn:disabled {
  border: none;
}
</style>
