<template>
  <LoadIndicator :busy="isLoading">
    <div class="row my-2">
      <div class="col">
        <div class="d-flex justify-content-end">
          <div class="d-flex justify-content-center gap-2">
            <select id="limit" class="form-select border-info rounded-3" v-model="filterLimit" @change="fetch">
              <option value="" style="font-weight: bold" disabled>{{ t("filter.limit") }}</option>
              <option v-for="limit in limits" :key="limit" :value="limit">{{ limit }}</option>
            </select>
            <button class="btn btn-info" @click="fetch">
              <i class="bi bi-arrow-clockwise"></i>
            </button>
          </div>
        </div>
        <button class="btn btn-info" @click="fetch" v-if="route.params.type === 'random'">
          <i class="bi bi-arrow-clockwise"></i>
        </button>
      </div>
    </div>
    <div class="row">
      <FillNotice v-if="recordings.length == 0">
        <h1>No results</h1>
      </FillNotice>
      <div v-for="recording in recordings" :key="recording.filename" class="mb-3 col-lg-6 col-xl-4 col-xxl-4 col-md-6 col-sm-8">
        <VideoItem :show-title="true" :recording="recording" @destroyed="destroyRecording" :show-selection="false" />
      </div>
    </div>
  </LoadIndicator>
</template>

<script setup lang="ts">
import VideoItem from "@/components/VideoItem.vue";
import type { DatabaseRecording as RecordingResponse } from "@/services/api/v1/MediaSinkClient";
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { createClient } from "@/services/api/v1/ClientFactory";
import LoadIndicator from "@/components/LoadIndicator.vue";
import { useI18n } from "vue-i18n";
import FillNotice from "@/components/FillNotice.vue";

const { t } = useI18n();

const route = useRoute();

const isLoading = ref(true);

watch(route, () => {
  fetch();
});

const filterLimit = (route.params.limit as string) || "25";
const limits = ref([25, 50, 100, 200, 500, 1000]);

const recordings = ref<RecordingResponse[]>([]);

const fetch = async () => {
  const client = createClient();
  const data = await client.videos.randomDetail({ limit: +filterLimit });
  recordings.value = data || [];
};

const destroyRecording = (recording: RecordingResponse) => {
  for (let i = 0; i < recordings.value.length; i += 1) {
    if (recordings.value[i]!.filename === recording.filename) {
      recordings.value.splice(i, 1);
      break;
    }
  }
};

onMounted(async () => {
  await fetch();
  isLoading.value = false;
});
</script>
