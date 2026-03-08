<template>
  <div class="justify-content-between d-flex gap-1">
    <JobStatus :jobs="jobs" :total-count="totalCount" @click="router.push('/jobs')" />
    <button v-if="!props.isRecording" class="btn btn-info d-flex gap-1" @click="emit('record', true)">
      <i class="bi bi-play-fill"></i>
      <span class="d-none d-sm-inline">{{ t("navtop.startRecording") }}</span>
    </button>
    <button v-else class="btn btn-danger blink d-flex gap-1" @click="emit('record', false)">
      <i class="bi bi-stop-fill"></i>
      <span class="d-none d-xl-inline d-flex justify-content-between">{{ t("navtop.stopRecording") }}</span>
    </button>
    <button class="btn btn-success d-flex gap-1 d-none d-sm-none d-md-block" @click="emit('add')">
      <i class="bi bi-plus-lg" />
      <span class="d-none d-xl-inline">{{ t("navtop.addStream") }}</span>
    </button>

    <button v-if="route.path.startsWith('/streams')" class="btn btn-lg btn-success shadow-sm border-success-subtle d-md-none position-fixed d-flex justify-content-center align-items-center" style="bottom: 5%; right: 10%; border-radius: 50%; width: 3.5rem; height: 3.5rem" @click="emit('add')">
      <i class="bi bi-plus-lg fw-bold"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import JobStatus from "./JobStatus.vue";
import type { DatabaseJob } from "../services/api/v1/MediaSinkClient";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const emit = defineEmits<{
  (e: "add"): void;
  (e: "record", value: boolean): void;
}>();

const props = defineProps<{
  jobs: DatabaseJob[];
  isRecording: boolean;
  totalCount: number;
}>();
</script>

<style scoped></style>
