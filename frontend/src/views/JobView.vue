<template>
  <div>
    <ModalConfirmDialog :show="showConfirmToggleWorkerDialog" @cancel="showConfirmToggleWorkerDialog = false" @confirm="toggleWorker">
      <template v-slot:header>
        <div class="d-flex justify-content-between">
          <h5 class="modal-title">
            <span v-if="processingJobs">Pause job worker?</span>
            <span v-else>Resume job worker?</span>
          </h5>
        </div>
      </template>
      <template v-slot:body>
        <span v-if="processingJobs">Do you want to pause the job worker?</span>
        <span v-else>Do you want to resume the job worker?</span>
      </template>
    </ModalConfirmDialog>
    <div class="row mb-2">
      <div class="col">
        <div class="d-flex justify-content-center gap-2">
          <div>
            <select class="form-select border-info rounded-3" v-model="tab">
              <option class="fw-bold" disabled readonly value="">Job types</option>
              <option v-for="tab in tabs" :value="tab.key" :key="tab.key">
                {{ tab.value }}
              </option>
            </select>
          </div>

          <div class="d-flex justify-content-center">
            <div class="col-auto">
              <button type="button" class="btn" :class="{ 'btn-success': !processingJobs, 'btn-danger rounded-3': processingJobs }" @click="showConfirmToggleWorkerDialog = true">
                <span v-if="processingJobs">
                  <i class="bi bi-pause-fill blink" />
                  <span class="ms-1 d-none d-sm-inline-flex">Pause</span>
                </span>
                <span v-else>
                  <i class="bi bi-play-fill" />
                  <span class="ms-1 d-none d-sm-inline-flex">Resume</span>
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="table-responsive">
      <DataTable :columns="columns" :data="jobs as unknown as TableRow[]" :pageSize="pageSize" :showPagination="true" :showPageSizeSelector="true" @page-change="handlePageChange" @page-size-change="handlePageSizeChange">
        <template v-slot:cell-active="{ row }">
          <span v-if="row.active" style="width: 1rem; height: 1rem" class="spinner-border text-success" role="status">
            <span class="visually-hidden">{{ t("jobTable.loading") }}</span>
          </span>
          <span v-else></span>
        </template>

        <template #cell-task="{ value }">
          <span class="badge bg-warning">{{ value }}</span>
        </template>

        <template #cell-channelName="{ value, row }">
          <div class="d-flex gap-1 justify-content-between fs-6">
            <RouterLink :to="`/channel/${row.channelId}/${row.channelName}`">
              {{ value }}
            </RouterLink>
            <RouterLink :to="'/recordings/' + row.recordingId">
              <i class="bi bi-film"></i>
            </RouterLink>
          </div>
        </template>

        <template v-slot:cell-command="{ value }">
          <input type="text" class="form-control form-control-sm border-primary-subtle" style="font-size: 0.8rem; font-family: monospace" disabled :value="value" />
        </template>

        <template #cell-progress="{ row }">
          <div v-if="row.progress" class="progress border border-primary-subtle">
            <div class="progress-bar progress-bar-striped bg-info progress-bar-animated" role="progressbar" :style="'width:' + row.progress + '%'" :aria-valuenow="(row as unknown as JobTableItem).progress || 0" aria-valuemin="0" :aria-valuemax="100"></div>
          </div>
          <span v-else>-</span>
        </template>

        <template #cell-createdAtFromNow="{ value }">
          <span class="text-ellipsis text-nowrap">{{ value }}</span>
        </template>

        <template #cell-info="{ value }">
          <!-- eslint-disable-next-line vue/no-textarea-mustache -->
          <textarea class="form-control">{{ value }}</textarea>
        </template>

        <template #cell-priority="{ value }">
          <div class="text-ellipsis badge text-nowrap text-white d-block m-0 p-2" :class="{ 'bg-success': value === 1, 'bg-warning': value === 2, 'bg-danger': (value as unknown as number) > 2 }">{{ getPrio(value as unknown as number) }}</div>
        </template>

        <template #cell-wokingDuration="{ value }">
          <span class="text-ellipsis text-nowrap">{{ value }}</span>
        </template>

        <template #cell-startedFromNow="{ value }">
          <span class="text-ellipsis text-nowrap">{{ value }}</span>
        </template>

        <template #cell-completedAtFromNow="{ value }">
          <span class="text-ellipsis text-nowrap">{{ value }}</span>
        </template>

        <template v-slot:cell-destroy="{ row }">
          <div class="btn-group-sm btn-group w-100">
            <button class="btn btn-outline-danger btn-sm" @click="destroy((row as unknown as JobTableItem).jobId)">Destroy</button>
          </div>
        </template>
      </DataTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { DbJobOrder, DbJobStatus } from "../services/api/v2/MediaSinkClient";
import { useJobStore } from "../stores/job";
import ModalConfirmDialog from "../components/modals/ModalConfirmDialog.vue";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { createClient } from "../services/api/v2/ClientFactory";
import DataTable from "../components/DataTable.vue";
import type { Column, TableRow } from "../types/table";
import { useI18n } from "vue-i18n";
import { type DbJob } from "../services/api/v2/MediaSinkClient";

const jobStore = useJobStore();
const route = useRoute();
const router = useRouter();
const processingJobs = ref(true);
const showConfirmToggleWorkerDialog = ref(false);
const tab = ref<JobType>(route.params.tab as JobType);

const tabs: { key: JobType; value: string }[] = [
  { key: "processing", value: "Processing" },
  { key: "open", value: "Open" },
  { key: "completed", value: "Completed" },
  { key: "canceled", value: "Canceled" },
  { key: "error", value: "Errors" },
];

watch(tab, (tab: JobType) => {
  router.push(`/jobs/${tab}`);
});

type JobType = "open" | "processing" | "completed" | "canceled" | "error";

const tabToJobType: { [key: string]: DbJobStatus } = {
  open: DbJobStatus.StatusJobOpen,
  processing: DbJobStatus.StatusJobOpen,
  completed: DbJobStatus.StatusJobCompleted,
  canceled: DbJobStatus.StatusJobCanceled,
  error: DbJobStatus.StatusJobError,
};

const jobs = computed(() => jobStore.withTime);

export type JobTableItem = DbJob & {
  createdAtFromNow: string;
  startedFromNow?: string;
  completedAtFromNow?: string;
  wokingDuration: string;
};

const { t } = useI18n();

const pageSize = 10; // Adjust as needed

// Define columns with width properties
const columns = computed<Column[]>(() => {
  if (tab.value === "processing") {
    return [
      { key: "channelName", label: t("jobTable.col.channel"), sortable: true, type: "string", width: "5%" },
      { key: "priority", label: t("jobTable.col.load"), sortable: true, type: "string", width: "5%" },
      //{ key: "filename", label: t("jobTable.col.file"), sortable: true, type: "string", width: "5%" },
      { key: "task", label: t("jobTable.col.task"), sortable: true, type: "string", width: "5%", class: "d-none d-lg-table-cell" },
      { key: "info", label: t("jobTable.col.info"), sortable: true, type: "string", width: "5%", class: "d-none d-lg-table-cell" },
      { key: "command", label: "Command", sortable: false, type: "string", width: "15%", class: "d-none d-lg-table-cell" },
      { key: "progress", label: t("jobTable.col.progress"), sortable: false, type: "string", width: "10%" },
      { key: "startedFromNow", label: t("jobTable.col.startedAt"), sortable: false, type: "string", width: "5%" },
      { key: "wokingDuration", label: t("jobTable.col.duration"), sortable: true, type: "number", width: "5%" },
      { key: "destroy", label: t("jobTable.col.destroy"), sortable: false, type: "string", width: "5%" },
    ];
  }
  if (tab.value === "canceled" || tab.value === "error") {
    return [
      { key: "channelName", label: t("jobTable.col.channel"), sortable: true, type: "string", width: "5%" },
      { key: "priority", label: t("jobTable.col.load"), sortable: true, type: "string", width: "5%" },
      { key: "task", label: t("jobTable.col.task"), sortable: true, type: "string", width: "5%", class: "d-none d-lg-table-cell" },
      { key: "info", label: t("jobTable.col.info"), sortable: true, type: "string", width: "", class: "d-none d-lg-table-cell" },
      { key: "command", label: "Command", sortable: false, type: "string", width: "15%", class: "d-none d-lg-table-cell" },
    ];
  }
  if (tab.value === "completed") {
    return [
      { key: "channelName", label: t("jobTable.col.channel"), sortable: true, type: "string", width: "15%" },
      { key: "priority", label: t("jobTable.col.load"), sortable: true, type: "string", width: "5%" },
      { key: "task", label: t("jobTable.col.task"), sortable: true, type: "string", width: "5%", class: "d-none d-lg-table-cell" },
      { key: "command", label: "Command", sortable: false, type: "string", width: "", class: "d-none d-lg-table-cell" },
      { key: "startedFromNow", label: t("jobTable.col.startedAt"), sortable: false, type: "string", width: "10%" },
      { key: "completedAtFromNow", label: t("jobTable.col.completedAt"), sortable: false, type: "string", width: "10%" },
      { key: "wokingDuration", label: t("jobTable.col.duration"), sortable: true, type: "number", width: "10%" },
    ];
  }
  return [
    { key: "channelName", label: t("jobTable.col.channel"), sortable: true, type: "string", width: "5%" },
    { key: "priority", label: t("jobTable.col.load"), sortable: true, type: "string", width: "5%" },
    //{ key: "filename", label: t("jobTable.col.file"), sortable: true, type: "string", width: "5%" },
    { key: "task", label: t("jobTable.col.task"), sortable: true, type: "string", width: "5%", class: "d-none d-lg-table-cell" },
    { key: "info", label: t("jobTable.col.info"), sortable: true, type: "string", width: "5%", class: "d-none d-lg-table-cell" },
    { key: "command", label: "Command", sortable: false, type: "string", width: "15%", class: "d-none d-lg-table-cell" },
    { key: "progress", label: t("jobTable.col.progress"), sortable: false, type: "string", width: "10%" },
    { key: "createdAtFromNow", label: t("jobTable.col.createdAt"), sortable: false, type: "string", width: "5%" },
    { key: "startedFromNow", label: t("jobTable.col.startedAt"), sortable: false, type: "string", width: "5%" },
    { key: "completedAtFromNow", label: t("jobTable.col.completedAt"), sortable: false, type: "string", width: "5%" },
    { key: "wokingDuration", label: t("jobTable.col.duration"), sortable: true, type: "number", width: "5%" },
    { key: "destroy", label: t("jobTable.col.destroy"), sortable: false, type: "string", width: "5%" },
  ];
});

const getPrio = (prio: number): string => {
  if (prio === 1) return "Low";
  if (prio === 2) return "Medium";
  if (prio > 2) return "High";
  return "bg-primary";
};

// Handlers for pagination events
const handlePageChange = (page: number) => {
  console.log(page);
};

const handlePageSizeChange = (size: number) => {
  console.log(size);
};

onMounted(async () => {
  await load();
});

const load = async () => {
  const activeLookup: { [key: string]: boolean | undefined } = {
    processing: true,
    open: false,
  };

  await jobStore.load([tabToJobType[tab.value]], DbJobOrder.JobOrderDESC, activeLookup[tab.value]);
};

const destroy = (id: number) => {
  if (window.confirm("Delete?")) {
    const client = createClient();
    client.jobs
      .jobsDelete({ id })
      .then(() => jobStore.destroy(id))
      .catch((res: any) => alert(res));
  }
};

const toggleWorker = () => {
  const client = createClient();
  const fn = processingJobs.value ? client.jobs.pauseCreate : client.jobs.resumeCreate;

  fn()
    .then(() => {
      processingJobs.value = !processingJobs.value;
    })
    .catch((res: any) => alert((<{ error: string }>res).error))
    .finally(() => (showConfirmToggleWorkerDialog.value = false));
};
</script>

<style scoped>
.job-table * {
  text-overflow: ellipsis;
  overflow: hidden;
  white-space: nowrap;
}

.job-table th {
  font-size: 0.9rem;
}
</style>
