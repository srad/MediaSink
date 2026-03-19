<template>
  <div class="admin-view container-fluid py-2 py-xl-3">
    <section class="card admin-hero border-0 shadow-sm mb-3">
      <div class="card-body p-3">
        <div class="row g-2 align-items-lg-end">
          <div class="col-24 col-xl-16">
            <h1 class="hero-title mb-2">{{ t("admin.title") }}</h1>
            <p class="hero-subtitle mb-0">{{ t("admin.subtitle") }}</p>
          </div>

          <div class="col-24 col-xl-8">
            <div class="row g-2 justify-content-xl-end">
              <div class="col-24 col-sm-12">
                <div class="status-chip h-100" :class="statusClass(importState)">
                  <div class="d-flex align-items-center gap-2">
                    <i class="bi bi-download"></i>
                    <span>{{ t("admin.importLabel") }}</span>
                  </div>
                  <strong>{{ statusLabel(importState) }}</strong>
                </div>
              </div>

              <div class="col-24 col-sm-12">
                <div class="status-chip h-100" :class="statusClass(previewState)">
                  <div class="d-flex align-items-center gap-2">
                    <i class="bi bi-images"></i>
                    <span>{{ t("admin.previewsLabel") }}</span>
                  </div>
                  <strong>{{ statusLabel(previewState) }}</strong>
                </div>
              </div>

              <div class="col-24 col-sm-12">
                <div class="status-chip h-100" :class="statusClass(metadataState)">
                  <div class="d-flex align-items-center gap-2">
                    <i class="bi bi-arrow-repeat"></i>
                    <span>{{ t("admin.metadataLabel") }}</span>
                  </div>
                  <strong>{{ statusLabel(metadataState) }}</strong>
                </div>
              </div>

              <div class="col-24 col-sm-12">
                <div class="status-chip h-100" :class="statusClass(chapterState)">
                  <div class="d-flex align-items-center gap-2">
                    <i class="bi bi-diagram-3"></i>
                    <span>{{ t("admin.chaptersLabel") }}</span>
                  </div>
                  <strong>{{ statusLabel(chapterState) }}</strong>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <div class="row g-3">
      <div class="col-24 col-xl-10">
        <section class="card admin-shell border-0 shadow-sm h-100">
          <div class="card-header bg-transparent border-0 p-3 pb-0">
            <h2 class="section-title mb-1">{{ t("admin.statusTitle") }}</h2>
            <p class="section-copy mb-0">{{ t("admin.statusSubtitle") }}</p>
          </div>

          <div class="card-body p-3">
            <div class="row g-2">
              <div class="col-24">
                <div class="panel-block h-100">
                  <div class="section-label mb-3">{{ t("admin.versionsTitle") }}</div>
                  <dl class="row g-2 mb-0 version-list">
                    <dt class="col-10">{{ t("admin.clientVersion") }}</dt>
                    <dd class="col-14 text-end">{{ clientVersion }}</dd>

                    <dt class="col-10">{{ t("admin.clientRevision") }}</dt>
                    <dd class="col-14 text-end text-truncate" :title="clientBuild">{{ clientBuild }}</dd>

                    <dt class="col-10">{{ t("admin.serverVersion") }}</dt>
                    <dd class="col-14 text-end">{{ serverInfo?.version || "..." }}</dd>

                    <dt class="col-10">{{ t("admin.serverRevision") }}</dt>
                    <dd class="col-14 text-end text-truncate" :title="serverInfo?.commit || '...'">{{ serverInfo?.commit || "..." }}</dd>
                  </dl>
                </div>
              </div>

              <div class="col-24 col-md-12 col-xl-24">
                <div class="panel-block h-100">
                  <div class="d-flex justify-content-between align-items-start gap-3 mb-3">
                    <div>
                      <h3 class="block-title mb-1">{{ t("admin.importLabel") }}</h3>
                      <p class="block-copy mb-0">{{ t("admin.startImportBody") }}</p>
                    </div>
                    <span class="status-badge" :class="statusClass(importState)">{{ statusLabel(importState) }}</span>
                  </div>

                  <div class="progress-meta">
                    <span>{{ importProgressValue }}/{{ importSizeValue }}</span>
                    <span>{{ importPercent }}%</span>
                  </div>
                  <div class="progress status-progress">
                    <div class="progress-bar" role="progressbar" :style="{ width: `${importPercent}%` }" :aria-valuenow="importPercent" aria-valuemin="0" aria-valuemax="100"></div>
                  </div>
                </div>
              </div>

              <div class="col-24 col-md-12 col-xl-24">
                <div class="panel-block h-100">
                  <div class="d-flex justify-content-between align-items-start gap-3 mb-3">
                    <div>
                      <h3 class="block-title mb-1">{{ t("admin.previewsLabel") }}</h3>
                      <p class="block-copy mb-0">{{ t("admin.regeneratePreviewsBody") }}</p>
                    </div>
                    <span class="status-badge" :class="statusClass(previewState)">{{ statusLabel(previewState) }}</span>
                  </div>

                  <div class="progress-meta">
                    <span>{{ previewCurrentValue }}/{{ previewTotalValue }}</span>
                    <span>{{ previewPercent }}%</span>
                  </div>
                  <div class="progress status-progress mb-2">
                    <div class="progress-bar bg-warning" role="progressbar" :style="{ width: `${previewPercent}%` }" :aria-valuenow="previewPercent" aria-valuemin="0" aria-valuemax="100"></div>
                  </div>
                  <div class="current-video text-truncate" :title="previewCurrentVideo">
                    {{ t("admin.currentVideo") }}: {{ previewCurrentVideo }}
                  </div>
                </div>
              </div>

              <div class="col-24 col-md-12 col-xl-24">
                <div class="panel-block h-100">
                  <div class="d-flex justify-content-between align-items-start gap-3">
                    <div>
                      <h3 class="block-title mb-1">{{ t("admin.metadataLabel") }}</h3>
                      <p class="block-copy mb-0">{{ t("admin.updateMetadataBody") }}</p>
                    </div>
                    <span class="status-badge" :class="statusClass(metadataState)">{{ statusLabel(metadataState) }}</span>
                  </div>
                </div>
              </div>

              <div class="col-24 col-md-12 col-xl-24">
                <div class="panel-block h-100">
                  <div class="d-flex justify-content-between align-items-start gap-3 mb-3">
                    <div>
                      <h3 class="block-title mb-1">{{ t("admin.chaptersLabel") }}</h3>
                      <p class="block-copy mb-0">{{ t("admin.regenerateChaptersBody") }}</p>
                    </div>
                    <span class="status-badge" :class="statusClass(chapterState)">{{ statusLabel(chapterState) }}</span>
                  </div>

                  <div v-if="lastChapterRun" class="row g-2">
                    <div class="col-24 col-sm-8">
                      <div class="summary-cell h-100">
                        <span>{{ t("admin.removedJobs") }}</span>
                        <strong>{{ lastChapterRun.removedJobs }}</strong>
                      </div>
                    </div>
                    <div class="col-24 col-sm-8">
                      <div class="summary-cell h-100">
                        <span>{{ t("admin.enqueuedJobs") }}</span>
                        <strong>{{ lastChapterRun.enqueued }}</strong>
                      </div>
                    </div>
                    <div class="col-24 col-sm-8">
                      <div class="summary-cell h-100">
                        <span>{{ t("admin.recordings") }}</span>
                        <strong>{{ lastChapterRun.recordings }}</strong>
                      </div>
                    </div>
                  </div>
                  <p v-else class="empty-copy mb-0">{{ t("admin.noChapterRun") }}</p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <div class="col-24 col-xl-14">
        <section class="card admin-shell border-0 shadow-sm h-100">
          <div class="card-header bg-transparent border-0 p-3 pb-0">
            <h2 class="section-title mb-1">{{ t("admin.actionsTitle") }}</h2>
            <p class="section-copy mb-0">{{ t("admin.actionsSubtitle") }}</p>
          </div>

          <div class="card-body p-3">
            <div class="row g-2">
              <div class="col-24 col-md-12">
                <article class="card action-card action-card-primary border-0 h-100">
                  <div class="card-body p-3 d-flex flex-column gap-2">
                    <div class="action-icon">
                      <i class="bi bi-arrow-repeat"></i>
                    </div>
                    <div class="action-body">
                      <h3>{{ t("admin.updateMetadataTitle") }}</h3>
                      <p>{{ t("admin.updateMetadataBody") }}</p>
                    </div>
                    <div class="action-footer mt-auto">
                      <span class="status-badge" :class="statusClass(metadataState)">{{ statusLabel(metadataState) }}</span>
                      <button class="btn btn-primary" :disabled="isAnyActionPending" @click="openConfirm('metadata')">
                        <span v-if="activeAction === 'metadata'" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        {{ t("admin.updateMetadataAction") }}
                      </button>
                    </div>
                  </div>
                </article>
              </div>

              <div class="col-24 col-md-12">
                <article class="card action-card action-card-warning border-0 h-100">
                  <div class="card-body p-3 d-flex flex-column gap-2">
                    <div class="action-icon">
                      <i class="bi bi-images"></i>
                    </div>
                    <div class="action-body">
                      <h3>{{ t("admin.regeneratePreviewsTitle") }}</h3>
                      <p>{{ t("admin.regeneratePreviewsBody") }}</p>
                    </div>
                    <div class="action-footer mt-auto">
                      <span class="status-badge" :class="statusClass(previewState)">{{ statusLabel(previewState) }}</span>
                      <button class="btn btn-warning" :disabled="isAnyActionPending" @click="openConfirm('previews')">
                        <span v-if="activeAction === 'previews'" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        {{ t("admin.regeneratePreviewsAction") }}
                      </button>
                    </div>
                  </div>
                </article>
              </div>

              <div class="col-24 col-md-12">
                <article class="card action-card action-card-success border-0 h-100">
                  <div class="card-body p-3 d-flex flex-column gap-2">
                    <div class="action-icon">
                      <i class="bi bi-folder-symlink"></i>
                    </div>
                    <div class="action-body">
                      <h3>{{ t("admin.startImportTitle") }}</h3>
                      <p>{{ t("admin.startImportBody") }}</p>
                    </div>
                    <div class="action-footer mt-auto">
                      <span class="status-badge" :class="statusClass(importState)">{{ statusLabel(importState) }}</span>
                      <button class="btn btn-success" :disabled="isAnyActionPending" @click="openConfirm('import')">
                        <span v-if="activeAction === 'import'" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        {{ t("admin.startImportAction") }}
                      </button>
                    </div>
                  </div>
                </article>
              </div>

              <div class="col-24 col-md-12">
                <article class="card action-card action-card-danger border-0 h-100">
                  <div class="card-body p-3 d-flex flex-column gap-2">
                    <div class="action-icon">
                      <i class="bi bi-diagram-3"></i>
                    </div>
                    <div class="action-body">
                      <h3>{{ t("admin.regenerateChaptersTitle") }}</h3>
                      <p>{{ t("admin.regenerateChaptersBody") }}</p>
                    </div>
                    <div class="action-footer mt-auto">
                      <span class="status-badge" :class="statusClass(chapterState)">{{ statusLabel(chapterState) }}</span>
                      <button class="btn btn-danger" :disabled="isAnyActionPending" @click="openConfirm('chapters')">
                        <span v-if="activeAction === 'chapters'" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        {{ t("admin.regenerateChaptersAction") }}
                      </button>
                    </div>
                  </div>
                </article>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>

    <ModalConfirmDialog :show="confirmAction !== null" @confirm="runConfirmedAction" @cancel="confirmAction = null">
      <template #header>
        <h5 class="mb-0">{{ confirmTitle }}</h5>
      </template>
      <template #body>
        <p class="mb-0">{{ confirmBody }}</p>
      </template>
    </ModalConfirmDialog>

  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, ref } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  ContentType,
  type ResponsesImportInfoResponse,
  type ResponsesServerInfoResponse,
  type ServicesRegenerationProgress,
} from "@/services/api/v2/MediaSinkClient";
import ModalConfirmDialog from "@/components/modals/ModalConfirmDialog.vue";
import { createClient } from "@/services/api/v2/ClientFactory";
import { useToastStore } from "@/stores/toast";

type ActionKind = "metadata" | "previews" | "import" | "chapters";
type StatusTone = "idle" | "running" | "queued";

type ChapterRegenerationResult = {
  removedJobs: number;
  enqueued: number;
  recordings: number;
};

const { t } = useI18n();
const toastStore = useToastStore();

const injectedVersion = inject("version");
const injectedBuild = inject("build");

const clientVersion = typeof injectedVersion === "string" ? injectedVersion : "unknown";
const clientBuild = typeof injectedBuild === "string" ? injectedBuild : "unknown";

const importInfo = ref<ResponsesImportInfoResponse>({
  isImporting: false,
  progress: 0,
  size: 0,
});
const previewsProgress = ref<ServicesRegenerationProgress>({
  current: 0,
  total: 0,
  currentVideo: "",
  isRunning: false,
});
const serverInfo = ref<ResponsesServerInfoResponse | null>(null);
const metadataUpdating = ref(false);
const activeAction = ref<ActionKind | null>(null);
const confirmAction = ref<ActionKind | null>(null);
const lastChapterRun = ref<ChapterRegenerationResult | null>(null);
const pollId = ref<number | null>(null);

const isImporting = computed(() => !!importInfo.value.isImporting);
const importProgressValue = computed(() => importInfo.value.progress ?? 0);
const importSizeValue = computed(() => importInfo.value.size ?? 0);
const isRegeneratingPreviews = computed(() => !!previewsProgress.value.isRunning);
const previewCurrentValue = computed(() => previewsProgress.value.current ?? 0);
const previewTotalValue = computed(() => previewsProgress.value.total ?? 0);
const previewCurrentVideo = computed(() => previewsProgress.value.currentVideo || "...");

const importPercent = computed(() => {
  if (!importSizeValue.value) return 0;
  return Math.min(100, Math.round((importProgressValue.value / importSizeValue.value) * 100));
});

const previewPercent = computed(() => {
  if (!previewTotalValue.value) return 0;
  return Math.min(100, Math.round((previewCurrentValue.value / previewTotalValue.value) * 100));
});

const importState = computed<StatusTone>(() => (isImporting.value ? "running" : "idle"));
const previewState = computed<StatusTone>(() => (isRegeneratingPreviews.value ? "running" : "idle"));
const metadataState = computed<StatusTone>(() => (metadataUpdating.value || activeAction.value === "metadata" ? "running" : "idle"));
const chapterState = computed<StatusTone>(() => {
  if (activeAction.value === "chapters") return "running";
  if (lastChapterRun.value) return "queued";
  return "idle";
});

const isAnyActionPending = computed(() => activeAction.value !== null);

const confirmTitle = computed(() => {
  switch (confirmAction.value) {
    case "import":
      return t("admin.confirmImportTitle");
    case "previews":
      return t("admin.confirmPreviewsTitle");
    case "metadata":
      return t("admin.confirmMetadataTitle");
    case "chapters":
      return t("admin.confirmChaptersTitle");
    default:
      return "";
  }
});

const confirmBody = computed(() => {
  switch (confirmAction.value) {
    case "import":
      return t("admin.confirmImportBody");
    case "previews":
      return t("admin.confirmPreviewsBody");
    case "metadata":
      return t("admin.confirmMetadataBody");
    case "chapters":
      return t("admin.confirmChaptersBody");
    default:
      return "";
  }
});

const statusClass = (state: StatusTone) => ({
  "status-running": state === "running",
  "status-queued": state === "queued",
  "status-idle": state === "idle",
});

const statusLabel = (state: StatusTone) => {
  switch (state) {
    case "running":
      return t("admin.running");
    case "queued":
      return t("admin.queued");
    default:
      return t("admin.idle");
  }
};

const formatError = (error: unknown): string => {
  if (typeof error === "string") return error;
  if (error && typeof error === "object") {
    if ("error" in error && typeof error.error === "string") {
      return error.error;
    }
    if ("message" in error && typeof error.message === "string") {
      return error.message;
    }
  }
  return "Unknown error";
};

const fetchDashboard = async (quiet = false) => {
  const client = createClient();

  try {
    const [importStateResponse, previewState, versionState, isUpdating] = await Promise.all([
      client.admin.importList(),
      client.previews.regenerateList(),
      client.admin.versionList(),
      client.videos.isupdatingCreate(),
    ]);

    importInfo.value = importStateResponse;
    previewsProgress.value = previewState;
    serverInfo.value = versionState;
    metadataUpdating.value = isUpdating;
  } catch (error) {
    if (!quiet) {
      toastStore.error({
        title: t("admin.dashboardErrorTitle"),
        message: formatError(error),
      });
    }
  }
};

const openConfirm = (action: ActionKind) => {
  if (isAnyActionPending.value) return;
  confirmAction.value = action;
};

const runConfirmedAction = async () => {
  const action = confirmAction.value;
  if (!action) return;

  confirmAction.value = null;
  activeAction.value = action;
  const client = createClient();

  try {
    switch (action) {
      case "import":
        await client.admin.importCreate();
        toastStore.success({
          title: t("admin.importStartedTitle"),
          message: t("admin.importStartedBody"),
        });
        break;
      case "previews":
        await client.previews.regenerateCreate();
        toastStore.success({
          title: t("admin.previewsStartedTitle"),
          message: t("admin.previewsStartedBody"),
        });
        break;
      case "metadata":
        await client.videos.updateinfoCreate();
        toastStore.success({
          title: t("admin.metadataFinishedTitle"),
          message: t("admin.metadataFinishedBody"),
        });
        break;
      case "chapters": {
        const result = await client.http.request<ChapterRegenerationResult, any>({
          path: "/admin/chapters/regenerate",
          method: "POST",
          type: ContentType.Json,
          format: "json",
        });
        lastChapterRun.value = result;
        toastStore.success({
          title: t("admin.chaptersQueuedTitle"),
          message: t("admin.chapterResultBody", {
            removedJobs: result.removedJobs,
            enqueued: result.enqueued,
            recordings: result.recordings,
          }),
        });
        break;
      }
    }
  } catch (error) {
    toastStore.error({
      title: t("admin.actionErrorTitle"),
      message: formatError(error),
    });
  } finally {
    activeAction.value = null;
    await fetchDashboard(true);
  }
};

const stopPolling = () => {
  if (pollId.value !== null) {
    window.clearInterval(pollId.value);
    pollId.value = null;
  }
};

onMounted(async () => {
  await fetchDashboard();
  pollId.value = window.setInterval(() => {
    void fetchDashboard(true);
  }, 5000);
});

onUnmounted(() => {
  stopPolling();
});

onBeforeRouteLeave(() => {
  stopPolling();
});
</script>

<style scoped lang="scss">
.admin-view {
  --admin-ink: #16324f;
  --admin-muted: #5e7287;
  --admin-surface: rgba(255, 255, 255, 0.78);
  --admin-line: rgba(22, 50, 79, 0.08);
  --admin-paper: linear-gradient(180deg, rgba(246, 249, 252, 0.98), rgba(237, 243, 249, 0.98));
}

.admin-hero,
.admin-shell {
  background: var(--admin-paper);
}

.hero-title {
  color: var(--admin-ink);
  font-size: clamp(1.45rem, 2vw, 2rem);
  line-height: 1.04;
}

.hero-subtitle,
.section-copy,
.block-copy,
.action-body p,
.current-video,
.empty-copy {
  color: var(--admin-muted);
}

.hero-subtitle {
  max-width: 36rem;
  font-size: 0.95rem;
}

.status-chip,
.panel-block,
.action-card {
  border: 1px solid rgba(255, 255, 255, 0.9);
  background: var(--admin-surface);
}

.status-chip {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.65rem;
  padding: 0.65rem 0.8rem;
  border-radius: 0.85rem;
  font-size: 0.84rem;
  font-weight: 600;
}

.status-chip strong,
.status-badge {
  white-space: nowrap;
}

.status-running {
  color: #0b5d3e;
  background: rgba(25, 135, 84, 0.12);
  border-color: rgba(25, 135, 84, 0.16);
}

.status-queued {
  color: #9a5a04;
  background: rgba(217, 119, 6, 0.12);
  border-color: rgba(217, 119, 6, 0.16);
}

.status-idle {
  color: var(--admin-ink);
  background: rgba(22, 50, 79, 0.05);
  border-color: rgba(22, 50, 79, 0.07);
}

.section-title {
  color: var(--admin-ink);
  font-size: 1rem;
  font-weight: 700;
}

.section-label {
  color: var(--admin-muted);
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.panel-block {
  height: 100%;
  padding: 0.85rem;
  border-radius: 0.85rem;
}

.version-list dt,
.version-list dd {
  margin-bottom: 0;
  font-size: 0.86rem;
}

.version-list dt {
  color: var(--admin-muted);
  font-weight: 600;
}

.version-list dd,
.block-title,
.action-body h3,
.summary-cell strong {
  color: var(--admin-ink);
}

.block-title,
.action-body h3 {
  font-size: 0.92rem;
  font-weight: 700;
}

.action-body h3 {
  margin-bottom: 0.35rem;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.35rem 0.6rem;
  border: 1px solid transparent;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.4rem;
  color: var(--admin-muted);
  font-size: 0.82rem;
}

.status-progress {
  height: 0.5rem;
  border-radius: 999px;
  background: rgba(22, 50, 79, 0.08);
}

.summary-cell {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  height: 100%;
  padding: 0.6rem 0.75rem;
  border-radius: 0.75rem;
  background: rgba(13, 110, 253, 0.06);
}

.summary-cell span {
  color: var(--admin-muted);
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.action-card {
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.78);
}

.action-card-primary {
  border-color: rgba(13, 110, 253, 0.12);
}

.action-card-warning {
  border-color: rgba(217, 119, 6, 0.12);
}

.action-card-success {
  border-color: rgba(25, 135, 84, 0.12);
}

.action-card-danger {
  border-color: rgba(194, 65, 12, 0.12);
}

.action-icon {
  width: 2.5rem;
  height: 2.5rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.8rem;
  color: white;
  font-size: 1rem;
  background: linear-gradient(135deg, rgba(13, 110, 253, 0.92), rgba(59, 130, 246, 0.72));
}

.action-card-warning .action-icon {
  background: linear-gradient(135deg, rgba(217, 119, 6, 0.92), rgba(245, 158, 11, 0.76));
}

.action-card-success .action-icon {
  background: linear-gradient(135deg, rgba(25, 135, 84, 0.92), rgba(38, 173, 113, 0.74));
}

.action-card-danger .action-icon {
  background: linear-gradient(135deg, rgba(194, 65, 12, 0.92), rgba(234, 88, 12, 0.78));
}

.action-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

@media (max-width: 575.98px) {
  .status-chip,
  .action-footer {
    flex-direction: column;
    align-items: flex-start;
  }

  .action-footer .btn {
    width: 100%;
  }
}
</style>
