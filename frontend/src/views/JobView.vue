<template>
  <div class="job-view">
    <ModalConfirmDialog :show="showConfirmToggleWorkerDialog" @cancel="showConfirmToggleWorkerDialog = false" @confirm="toggleWorker">
      <template #header>
        <h5 class="modal-title mb-0">
          {{ workerRunning ? "Pause job worker?" : "Resume job worker?" }}
        </h5>
      </template>
      <template #body>
        <span v-if="workerRunning">Queued jobs will stop moving until the worker is resumed.</span>
        <span v-else>Queued jobs will continue once the worker is resumed.</span>
      </template>
    </ModalConfirmDialog>

    <div class="job-bar">
      <div class="job-bar__left">
        <h1 class="job-title">Jobs</h1>
        <div class="job-tabs" role="tablist" aria-label="Job tabs">
          <button v-for="tabOption in tabs" :key="tabOption.key" type="button" class="job-tab" :class="{ 'job-tab--active': currentTab === tabOption.key }" @click="goToTab(tabOption.key)">
            {{ tabOption.label }}
          </button>
        </div>
      </div>

      <div class="job-bar__right">
        <span class="job-meta">{{ currentSummary }}</span>
        <label class="limit-control">
          <span>Rows</span>
          <select v-model.number="rowLimit" class="form-select form-select-sm">
            <option v-for="option in rowLimitOptions" :key="option" :value="option">
              {{ option === -1 ? "All" : option }}
            </option>
          </select>
        </label>
        <button type="button" class="btn btn-sm" :class="workerRunning ? 'btn-outline-danger' : 'btn-outline-success'" :disabled="togglingWorker" @click="showConfirmToggleWorkerDialog = true">
          {{ workerRunning ? "Pause" : "Resume" }}
        </button>
      </div>
    </div>

    <div class="job-panel">
      <div class="table-responsive">
        <table class="table align-middle mb-0 job-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>Task</th>
              <th>Channel</th>
              <th class="d-none d-lg-table-cell">Info</th>
              <th v-if="currentTab === 'active'">Progress</th>
              <th>{{ currentTab === "active" ? "Updated" : "Completed" }}</th>
              <th class="text-end">Action</th>
            </tr>
          </thead>
          <tbody v-if="pagedRows.length">
            <tr v-for="job in pagedRows" :key="job.jobId">
              <td>
                <span class="status-badge" :class="statusClass(job)">
                  {{ statusLabel(job) }}
                </span>
              </td>
              <td>
                <span class="task-label">{{ formatTask(job.task) }}</span>
              </td>
              <td>
                <div class="channel-cell">
                  <RouterLink :to="`/channel/${job.channelId}/${job.channelName}`">
                    {{ job.channelName }}
                  </RouterLink>
                  <RouterLink v-if="job.recordingId > 0" :to="`/recordings/${job.recordingId}`" aria-label="Open recording">
                    <i class="bi bi-film" />
                  </RouterLink>
                </div>
              </td>
              <td class="d-none d-lg-table-cell">
                <span class="cell-muted" :title="detailText(job)">{{ detailText(job) }}</span>
              </td>
              <td v-if="currentTab === 'active'">
                <div v-if="job.active" class="progress-cell">
                  <div class="progress">
                    <div class="progress-bar progress-bar-striped progress-bar-animated bg-info" role="progressbar" :style="{ width: `${formatProgress(job.progress)}%` }" :aria-valuenow="formatProgress(job.progress)" aria-valuemin="0" aria-valuemax="100"></div>
                  </div>
                  <span>{{ formatProgress(job.progress) }}%</span>
                </div>
                <span v-else class="cell-muted">Waiting</span>
              </td>
              <td>
                <span class="cell-muted">{{ timeLabel(job) }}</span>
              </td>
              <td class="text-end">
                <button type="button" class="btn btn-link btn-sm text-danger p-0" :disabled="isDestroying(job.jobId)" :aria-label="`Destroy job ${job.jobId}`" @click="destroy(job.jobId)">
                  <span v-if="isDestroying(job.jobId)">...</span>
                  <i v-else class="bi bi-trash3" />
                </button>
              </td>
            </tr>
          </tbody>
          <tbody v-else>
            <tr>
              <td :colspan="currentTab === 'active' ? 7 : 6" class="empty-cell">No jobs.</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="totalPages > 1" class="pagination-bar">
        <button type="button" class="pagination-nav" :disabled="currentPage === 1" @click="goToPage(currentPage - 1)">
          Prev
        </button>
        <div class="pagination-info">
          <button v-for="page in visiblePages" :key="page" type="button" class="page-chip" :class="{ 'page-chip--active': page === currentPage }" @click="goToPage(page)">
            {{ page }}
          </button>
        </div>
        <span class="pagination-summary">Page {{ currentPage }} / {{ totalPages }}</span>
        <button type="button" class="pagination-nav" :disabled="currentPage === totalPages" @click="goToPage(currentPage + 1)">
          Next
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { DbJobOrder, DbJobPriority, DbJobStatus, type DbJob } from "../services/api/v2/MediaSinkClient";
import { createClient } from "../services/api/v2/ClientFactory";
import ModalConfirmDialog from "../components/modals/ModalConfirmDialog.vue";
import { decorateJobsWithTime, type JobTableItem, useJobStore } from "../stores/job";

type JobTab = "active" | "completed" | "canceled" | "error";

const client = createClient();
const jobStore = useJobStore();
const route = useRoute();
const router = useRouter();

const tabs: { key: JobTab; label: string }[] = [
  { key: "active", label: "Active" },
  { key: "completed", label: "Completed" },
  { key: "error", label: "Errors" },
  { key: "canceled", label: "Canceled" },
];

const historyStateByTab: Record<Exclude<JobTab, "active">, DbJobStatus> = {
  completed: DbJobStatus.StatusJobCompleted,
  canceled: DbJobStatus.StatusJobCanceled,
  error: DbJobStatus.StatusJobError,
};

const ROW_LIMIT_STORAGE_KEY = "jobs-row-limit";

const loadRowLimit = (): number => {
  try {
    const value = Number.parseInt(localStorage.getItem(ROW_LIMIT_STORAGE_KEY) || "25", 10);
    return [10, 25, 50, 100, -1].includes(value) ? value : 25;
  } catch {
    return 25;
  }
};

const rowLimitOptions = [10, 25, 50, 100, -1];
const showConfirmToggleWorkerDialog = ref(false);
const workerRunning = ref(true);
const togglingWorker = ref(false);
const destroyingJobIds = ref<number[]>([]);
const historyJobs = ref<JobTableItem[]>([]);
const rowLimit = ref(loadRowLimit());
const currentPage = ref(1);

const normalizeTab = (tab: string | undefined): JobTab => {
  switch (tab) {
    case "completed":
    case "canceled":
    case "error":
    case "active":
      return tab;
    case "open":
    case "processing":
    default:
      return "active";
  }
};

const currentTab = computed<JobTab>(() => normalizeTab(route.params.tab as string | undefined));
const activeJobs = computed(() =>
  decorateJobsWithTime(
    [...jobStore.all].sort((a, b) => +b.active - +a.active || Date.parse(b.createdAt) - Date.parse(a.createdAt)),
  ),
);
const processingCount = computed(() => activeJobs.value.filter((job) => job.active).length);
const queuedCount = computed(() => activeJobs.value.filter((job) => !job.active).length);
const rows = computed<JobTableItem[]>(() => (currentTab.value === "active" ? activeJobs.value : historyJobs.value));
const pageSize = computed(() => (rowLimit.value === -1 ? Math.max(rows.value.length, 1) : rowLimit.value));
const totalPages = computed(() => Math.max(1, Math.ceil(rows.value.length / pageSize.value)));
const pagedRows = computed<JobTableItem[]>(() => {
  if (rowLimit.value === -1) {
    return rows.value;
  }

  const start = (currentPage.value - 1) * pageSize.value;
  return rows.value.slice(start, start + pageSize.value);
});
const currentSummary = computed(() => {
  if (currentTab.value === "active") {
    return `${processingCount.value} running · ${queuedCount.value} queued · ${workerRunning.value ? "worker on" : "worker paused"} · ${pagedRows.value.length}/${rows.value.length}`;
  }
  return `${pagedRows.value.length}/${rows.value.length} ${currentTab.value}`;
});
const visiblePages = computed<number[]>(() => {
  if (totalPages.value <= 5) {
    return Array.from({ length: totalPages.value }, (_, index) => index + 1);
  }

  const start = Math.max(1, currentPage.value - 2);
  const end = Math.min(totalPages.value, start + 4);
  const normalizedStart = Math.max(1, end - 4);
  return Array.from({ length: end - normalizedStart + 1 }, (_, index) => normalizedStart + index);
});

watch(rowLimit, (value) => {
  try {
    localStorage.setItem(ROW_LIMIT_STORAGE_KEY, String(value));
  } catch {
    console.error("[JobView] Failed to persist row limit");
  }

  currentPage.value = 1;
});

watch([rows, totalPages], () => {
  if (currentPage.value > totalPages.value) {
    currentPage.value = totalPages.value;
  }
});

watch(
  () => route.params.tab,
  async (rawTab) => {
    const normalizedTab = normalizeTab(rawTab as string | undefined);

    if (rawTab !== normalizedTab) {
      await router.replace(`/jobs/${normalizedTab}`);
      return;
    }

    await loadWorkerStatus();

    if (normalizedTab === "active") {
      historyJobs.value = [];
      return;
    }

    await loadHistory(normalizedTab);
  },
  { immediate: true },
);

const loadWorkerStatus = async () => {
  try {
    const response = await client.jobs.workerList();
    workerRunning.value = response.isProcessing;
  } catch (error) {
    console.error("[JobView] Failed to load worker status", error);
  }
};

const loadHistory = async (tab: Exclude<JobTab, "active">) => {
  const response = await client.jobs.listCreate({
    skip: 0,
    take: 100,
    states: [historyStateByTab[tab]],
    sortOrder: DbJobOrder.JobOrderDESC,
  });

  historyJobs.value = decorateJobsWithTime(response.jobs || []);
};

const goToTab = (tab: JobTab) => {
  router.push(`/jobs/${tab}`);
};

const goToPage = (page: number) => {
  currentPage.value = Math.max(1, Math.min(totalPages.value, page));
};

const formatTask = (task: string): string =>
  task
    .split("-")
    .map((part) => `${part.charAt(0).toUpperCase()}${part.slice(1)}`)
    .join(" ");

const formatProgress = (progress: string | number | undefined): number => {
  const parsed = typeof progress === "number" ? progress : Number.parseFloat(progress || "0");
  if (Number.isNaN(parsed)) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round(parsed)));
};

const priorityLabel = (priority: number): string => {
  if (priority <= DbJobPriority.PriorityHigh) {
    return "High";
  }
  if (priority <= DbJobPriority.PriorityNormal) {
    return "Normal";
  }
  return "Low";
};

const statusLabel = (job: JobTableItem): string => {
  if (currentTab.value === "active") {
    return job.active ? "Running" : "Queued";
  }
  if (job.status === DbJobStatus.StatusJobCompleted) {
    return "Done";
  }
  if (job.status === DbJobStatus.StatusJobError) {
    return "Error";
  }
  return "Canceled";
};

const statusClass = (job: JobTableItem): string => {
  if (currentTab.value === "active") {
    return job.active ? "status-badge--running" : "status-badge--queued";
  }
  if (job.status === DbJobStatus.StatusJobCompleted) {
    return "status-badge--done";
  }
  if (job.status === DbJobStatus.StatusJobError) {
    return "status-badge--error";
  }
  return "status-badge--canceled";
};

const detailText = (job: JobTableItem): string => job.info || job.filename || job.command || "-";

const timeLabel = (job: JobTableItem): string => {
  if (currentTab.value === "active") {
    return job.active ? (job.startedAt ? job.startedFromNow : job.createdAtFromNow) : job.createdAtFromNow;
  }
  return job.completedAtFromNow;
};

const isDestroying = (jobId: number): boolean => destroyingJobIds.value.includes(jobId);

const destroy = async (jobId: number) => {
  if (isDestroying(jobId) || !window.confirm("Delete job?")) {
    return;
  }

  destroyingJobIds.value = [...destroyingJobIds.value, jobId];

  try {
    await client.jobs.jobsDelete({ id: jobId });
    jobStore.destroy(jobId);
    historyJobs.value = historyJobs.value.filter((job) => job.jobId !== jobId);
  } catch (error) {
    const message = (error as { error?: string })?.error || "Failed to delete job.";
    alert(message);
  } finally {
    destroyingJobIds.value = destroyingJobIds.value.filter((id) => id !== jobId);
  }
};

const toggleWorker = async () => {
  togglingWorker.value = true;

  try {
    const fn = workerRunning.value ? client.jobs.pauseCreate : client.jobs.resumeCreate;
    await fn();
    workerRunning.value = !workerRunning.value;
  } catch (error) {
    const message = (error as { error?: string })?.error || "Failed to change worker state.";
    alert(message);
  } finally {
    togglingWorker.value = false;
    showConfirmToggleWorkerDialog.value = false;
  }
};
</script>

<style scoped>
.job-view {
  display: grid;
  gap: 0.75rem;
}

.job-bar,
.job-panel {
  border: 1px solid rgba(var(--bs-border-color-rgb), 0.9);
  border-radius: 0.85rem;
  background: var(--bs-body-bg);
}

.job-bar {
  padding: 0.8rem 0.95rem;
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}

.job-bar__left,
.job-bar__right {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  flex-wrap: wrap;
}

.job-title {
  margin: 0;
  font-size: 1rem;
}

.job-tabs {
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
}

.job-tab {
  border: 1px solid rgba(var(--bs-border-color-rgb), 0.9);
  background: transparent;
  color: var(--bs-body-color);
  border-radius: 999px;
  padding: 0.22rem 0.7rem;
  font-size: 0.82rem;
  font-weight: 600;
}

.job-tab--active {
  background: rgba(var(--bs-info-rgb), 0.12);
  border-color: rgba(var(--bs-info-rgb), 0.35);
  color: var(--bs-info-text-emphasis);
}

.job-meta {
  color: var(--bs-secondary-color);
  font-size: 0.84rem;
  white-space: nowrap;
}

.limit-control {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--bs-secondary-color);
  font-size: 0.82rem;
}

.limit-control select {
  min-width: 4.5rem;
}

.job-table {
  --bs-table-bg: transparent;
  margin-bottom: 0;
}

.job-table th,
.job-table td {
  padding: 0.7rem 0.95rem;
  vertical-align: middle;
}

.job-table th {
  color: var(--bs-secondary-color);
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.job-table td {
  font-size: 0.92rem;
}

.status-badge,
.task-label {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  font-size: 0.75rem;
  font-weight: 700;
}

.status-badge--running {
  background: rgba(var(--bs-success-rgb), 0.12);
  color: var(--bs-success-text-emphasis);
}

.status-badge--queued {
  background: rgba(var(--bs-warning-rgb), 0.18);
  color: var(--bs-warning-text-emphasis);
}

.status-badge--done {
  background: rgba(var(--bs-secondary-rgb), 0.14);
  color: var(--bs-secondary-color);
}

.status-badge--error {
  background: rgba(var(--bs-danger-rgb), 0.12);
  color: var(--bs-danger-text-emphasis);
}

.status-badge--canceled {
  background: rgba(var(--bs-dark-rgb), 0.08);
  color: var(--bs-secondary-color);
}

.task-label {
  background: rgba(var(--bs-info-rgb), 0.1);
  color: var(--bs-info-text-emphasis);
}

.channel-cell {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  min-width: 0;
}

.channel-cell a:first-child,
.cell-muted {
  display: inline-block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.cell-muted {
  color: var(--bs-secondary-color);
}

.progress-cell {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  min-width: 10rem;
}

.progress-cell .progress {
  flex: 1 1 auto;
  height: 0.45rem;
}

.progress-cell span {
  width: 2.5rem;
  text-align: right;
  color: var(--bs-secondary-color);
  font-size: 0.82rem;
}

.empty-cell {
  padding: 1rem;
  color: var(--bs-secondary-color);
  text-align: center;
}

.pagination-bar {
  padding: 0.75rem 0.95rem 0.85rem;
  border-top: 1px solid rgba(var(--bs-border-color-rgb), 0.6);
  display: flex;
  gap: 0.75rem;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.pagination-info {
  display: flex;
  gap: 0.3rem;
  align-items: center;
}

.pagination-nav {
  border: 1px solid rgba(var(--bs-border-color-rgb), 0.95);
  background: rgba(var(--bs-body-bg-rgb), 0.98);
  color: var(--bs-body-color);
  border-radius: 999px;
  min-width: 3.6rem;
  height: 2rem;
  padding: 0 0.8rem;
  font-size: 0.82rem;
}

.pagination-nav:hover:not(:disabled) {
  background: rgba(var(--bs-info-rgb), 0.08);
  border-color: rgba(var(--bs-info-rgb), 0.3);
  color: var(--bs-body-color);
}

.pagination-nav:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.page-chip {
  border: 1px solid rgba(var(--bs-border-color-rgb), 0.9);
  background: transparent;
  color: var(--bs-body-color);
  border-radius: 999px;
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.55rem;
  font-size: 0.8rem;
}

.page-chip--active {
  background: rgba(var(--bs-info-rgb), 0.12);
  border-color: rgba(var(--bs-info-rgb), 0.35);
  color: var(--bs-info-text-emphasis);
}

.pagination-summary {
  color: var(--bs-secondary-color);
  font-size: 0.82rem;
}

@media (max-width: 767.98px) {
  .job-bar {
    padding: 0.75rem 0.85rem;
  }

  .job-meta {
    white-space: normal;
  }

  .job-table th,
  .job-table td {
    padding: 0.65rem 0.75rem;
  }

  .pagination-bar {
    padding: 0.7rem 0.85rem 0.8rem;
    justify-content: space-between;
  }
}
</style>
