<template>
  <div>
    <div class="d-flex justify-content-center align-items-center mb-3">
      <div>
        <select class="form-select" v-model="filterChannel">
          <option disabled readonly class="fw-bold">Filter channel</option>
          <option value="">Show All</option>
          <option :value="f" v-for="f in filter" :key="f">{{ f }}</option>
        </select>
      </div>
    </div>
    <div class="row">
      <FillNotice v-if="filteredVideos.length == 0">
        <h1><i class="bi bi-heartbreak-fill" style="color: deeppink"></i></h1>
      </FillNotice>
      <div v-for="recording in filteredVideos" :key="recording.filename" class="mb-3 col-lg-5 col-xl-4 col-xxl-4 col-md-10">
        <VideoItem :recording="recording" @destroyed="destroyRecording" @bookmark="bookmark" :show-selection="false" :show-title="false" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DatabaseRecording } from "../services/api/v1/MediaSinkClient";
import VideoItem from "../components/VideoItem.vue";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { createClient } from "../services/api/v1/ClientFactory";
import FillNotice from "@/components/FillNotice.vue";

// --------------------------------------------------------------------------------------
// Refs
// --------------------------------------------------------------------------------------

const { t } = useI18n();
const videos = ref<DatabaseRecording[]>([]);
const filterChannel = ref("");

// --------------------------------------------------------------------------------------
// Computes
// --------------------------------------------------------------------------------------

const filter = computed(() => Array.from(new Set(videos.value.map((x: DatabaseRecording) => x.channelName))));

const filteredVideos = computed(() => {
  if (filterChannel.value === "") {
    return videos.value;
  }
  return videos.value.filter((x) => x.channelName === filterChannel.value);
});

// --------------------------------------------------------------------------------------
// Functions
// --------------------------------------------------------------------------------------

const removeItem = (recording: DatabaseRecording) => {
  const i = videos.value.findIndex((r) => r.filename === recording.filename);
  if (i !== -1) {
    videos.value.splice(i, 1);
  }
};

const bookmark = (recording: DatabaseRecording) => {
  if (!recording.bookmark) {
    removeItem(recording);
  }
};

const destroyRecording = async (recording: DatabaseRecording) => {
  if (!window.confirm(t("crud.destroy", [recording.filename]))) {
    return;
  }

  try {
    const client = createClient();
    await client.videos.videosDelete({ id: recording.recordingId });
    removeItem(recording);
  } catch (ex) {
    alert(ex);
  }
};

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onMounted(async () => {
  const client = createClient();
  const data = await client.videos.bookmarksList();
  videos.value = (data || []) as DatabaseRecording[];
});
</script>
