<template>
  <div class="container-fluid py-2">
    <div class="btn-group">
      <button type="button" :disabled="isUpdating" class="btn btn-primary me-2" @click="updateInfo" :class="{ 'blink btn-danger': isUpdating }">
        <span class="me-2">Update video metadata</span>
        <i class="bi bi-arrow-clockwise"></i>
      </button>
      <button class="btn btn-primary me-2" @click="previews">
        <span class="me-2">Regenerate Previews</span>
        <i class="bi bi-card-image"></i>
      </button>

      <button :disabled="importing" class="btn btn-primary" @click="startImport">
        <span class="me-2">Start Import</span>
        <i class="bi bi-disc"></i>
      </button>
    </div>

    <div v-if="previewsProgress?.isRunning" class="d-flex gap-2">
      <h5>Generating previews</h5>
      <span :class="{ 'blink bg-danger-subtle': importing }">Generating ({{ previewsProgress.current }}/{{ previewsProgress.total }})</span>
      <div class="progress">
        <span class="progress-bar" role="progressbar" :style="{ width: (previewsProgress.current! / previewsProgress.total!) * 100 + '%' }" :aria-valuenow="previewsProgress.current" aria-valuemin="0" :aria-valuemax="previewsProgress.total"></span>
      </div>
    </div>

    <div v-if="importing" class="d-flex gap-2">
      <span :class="{ 'blink bg-danger-subtle': importing }">Importing ({{ importProgress }}/{{ importSize }})</span>
      <div class="progress">
        <span class="progress-bar" role="progressbar" :style="{ width: (importProgress / importSize) * 100 + '%' }" aria-valuenow="{{importProgress}}" aria-valuemin="0" aria-valuemax="{{importSize}}"></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { createClient } from "@/services/api/v1/ClientFactory";
import type { ServicesRegenerationProgress } from "@/services/api/v1/MediaSinkClient.ts";
import { mount } from "@vue/test-utils";

const importing = ref(false);
const importProgress = ref(0);
const importSize = ref(0);

const generatingPreviews = ref(false);
const previewsProgress = ref<ServicesRegenerationProgress | null>(null);
const previewPolling = ref<number | null>(null);

watch(generatingPreviews, (value) => {
  const client = createClient();
  if (value) {
    previewPolling.value = setInterval(async () => (previewsProgress.value = await client.previews.regenerateList()), 5000);
  } else {
    unpoll();
  }
});

const isUpdating = ref(false);

const id = ref<number>(0);

//const receivedMb = computed(() => ((netInfo.value?.receiveBytes || 0) / 1024 / 1024).toFixed(2));
//const transmittedMb = computed(() => ((netInfo.value?.transmitBytes || 0) / 1024 / 1024).toFixed(2));

const unpoll = () => {
  if (previewPolling.value) {
    clearInterval(previewPolling.value);
    previewPolling.value = null;
  }
};

const startImport = async () => {
  if (window.confirm("Start Import?")) {
    const client = createClient();
    await client.admin.importCreate();
    importing.value = true;
  }
};

const previews = async () => {
  if (window.confirm("Regenerate all previews?")) {
    const client = createClient();
    client.previews
      .regenerateCreate()
      .then(() => (generatingPreviews.value = true))
      .catch((res) => alert(res.error));
  }
};

const updateInfo = () => {
  if (window.confirm("Check all durations and update in database?")) {
    const client = createClient();
    client.videos
      .updateinfoCreate()
      .then(() => (isUpdating.value = true))
      .catch((res) => alert((<{ error: string }>res).error));
  }
};

onMounted(() => {
  const client = createClient();
  client.previews.regenerateList().then((res) => {
    if (res.isRunning) {
      previewsProgress.value = res;
    }
  });
});

onBeforeRouteLeave(() => {
  clearInterval(id.value);
  unpoll();
});
</script>

<style scoped></style>
