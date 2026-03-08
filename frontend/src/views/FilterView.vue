<template>
  <LoadIndicator :busy="isLoading">
    <div>
      <div class="row mb-2">
        <div class="col">
          <div class="d-flex justify-content-center">
            <!-- filter row -->
            <div class="d-flex justify-content-evenly gap-1">
              <select ref="filterColumnSelect" class="form-select border-info rounded-3" v-model="filterColumn" @change="filterChanged">
                <option value="" style="font-weight: bold" disabled>{{ t("filter.orderBy") }}</option>
                <option v-for="col in columns" :key="col[1]" :value="col[1]">{{ col[0] }}</option>
              </select>
              <select ref="sortOrderSelect" class="form-select text-capitalize border-info rounded-3" v-model="filterOrder" @input="filterChanged">
                <option value="" style="font-weight: bold" disabled>{{ t("filter.order") }}</option>
                <option v-for="o in order" :key="o" :value="o">{{ o }}</option>
              </select>
              <select ref="filterLimitSelect" id="limit" class="form-select border-info rounded-3" v-model="filterLimit" @change="filterChanged">
                <option value="" style="font-weight: bold" disabled>{{ t("filter.limit") }}</option>
                <option v-for="limit in limits" :key="limit" :value="limit">{{ limit }}</option>
              </select>

              <button type="button" class="btn btn-info rounded-3" @click="resetFilters">
                {{ t("filter.reset") }}
              </button>
            </div>
            <!-- filter row -->
            <button class="btn btn-primary btn-sm" @click="filterChanged" v-if="route.params.type === 'random'">Refresh</button>
          </div>
        </div>
      </div>
    </div>
    <div class="row">
      <FillNotice v-if="videos.length == 0">
        <h1>No results</h1>
      </FillNotice>
      <div v-for="video in videos" :key="video.video.recordingId" class="mb-3 col-lg-6 col-xl-6 col-xxl-4 col-md-8 col-sm-8">
        <VideoItem :job="video.jobTask" :show-title="true" :recording="video.video" @destroyed="destroyRecording" :show-selection="false" />
      </div>
    </div>
  </LoadIndicator>
</template>

<script setup lang="ts">
import { type DatabaseRecording, type DatabaseRecording as RecordingResponse, QueriesSortOrder, RequestsVideoSortColumn } from "../services/api/v1/MediaSinkClient";
import VideoItem from "../components/VideoItem.vue";
import { ref, useTemplateRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { createClient } from "../services/api/v1/ClientFactory";
import LoadIndicator from "../components/LoadIndicator.vue";
import { useJobStore } from "../stores/job";
import { useSettingsStore } from "../stores/settings";
import FillNotice from "@/components/FillNotice.vue";

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const jobStore = useJobStore();
const settingsStore = useSettingsStore();

const isLoading = ref(true);

const sortOrderSelect = useTemplateRef<HTMLSelectElement>("sortOrderSelect");
const filterColumnSelect = useTemplateRef<HTMLSelectElement>("filterColumnSelect");
const filterLimitSelect = useTemplateRef<HTMLSelectElement>("filterLimitSelect");

const filterOrder = ref("desc");
const filterColumn = ref("created_at");
const filterLimit = ref("100");

const fetch = async (column?: RequestsVideoSortColumn, order?: QueriesSortOrder, limit?: number) => {
  isLoading.value = true;
  const client = createClient();
  const data = await client.videos.filterCreate({
    skip: 0,
    take: limit || 100,
    sortColumn: column || RequestsVideoSortColumn.SortColumnCreatedAt,
    sortOrder: order || QueriesSortOrder.SortDesc,
  });
  videos.value =
    (data.videos ?? []).map((video: DatabaseRecording) => ({
      video: video,
      jobTask: jobStore.isProcessing(video.recordingId),
    })) || [];
  isLoading.value = false;
};

watch(
  [
    () => route.query.column,
    () => route.query.order,
    () => route.query.limit, // Watch limit directly too
  ],
  async () => {
    const column = route.query.column as RequestsVideoSortColumn;
    const order = route.query.order as QueriesSortOrder;
    const limit = parseInt(route.query.limit as string) || settingsStore.filterPageSize;
    await fetch(column, order, limit);
  },
  {
    immediate: true, // Set to true if you want the fetch function to run
    // immediately when the component mounts, using the
    // initial values from the URL query parameters.
    // This mimics the initial run behavior of watchEffect.
  },
);

const limits = [25, 50, 100, 200, 500, 1000];

const columns = [
  ["Created at", "created_at"],
  ["Filesize", "size"],
  ["Video duration", "duration"],
];

const order: string[] = ["asc", "desc"];

interface VideoResult {
  video: RecordingResponse;
  jobTask: string | null;
}

const videos = ref<VideoResult[]>([]);

const filterChanged = () => {
  const query = {
    order: sortOrderSelect.value?.value,
    column: filterColumnSelect.value?.value,
    limit: filterLimitSelect.value?.value,
  };

  if (query.limit) {
    settingsStore.setFilterViewPageSize(parseInt(query.limit));
  }

  router.replace({
    path: route.path,
    query,
    force: true,
  });
};

const resetFilters = () => {
  filterOrder.value = order[1]!;
  filterColumn.value = columns[0]![1]!;
  filterLimit.value = settingsStore.filterPageSize.toString();
  filterChanged();
};

const destroyRecording = (recording: RecordingResponse) => {
  for (let i = 0; i < videos.value.length; i += 1) {
    if (videos.value[i]!.video.filename === recording.filename) {
      videos.value.splice(i, 1);
      break;
    }
  }
};
</script>
