<template>
  <div class="px-3 pt-3">
    <!-- Tabs -->
    <ul class="nav nav-tabs mb-4">
      <li class="nav-item">
        <button class="nav-link px-4" :class="{ active: sim.tab === 'search' }" @click="sim.tab = 'search'">
          <i class="bi bi-search me-2"></i>Search
        </button>
      </li>
      <li class="nav-item">
        <button class="nav-link px-4" :class="{ active: sim.tab === 'group' }" @click="sim.tab = 'group'">
          <i class="bi bi-collection me-2"></i>Group
        </button>
      </li>
    </ul>

    <!-- ─── Analysis progress banner ─────────────────────────────────────── -->
    <div v-if="analysisJobCount > 0" class="analysis-progress-bar mb-4">
      <div class="d-flex align-items-center gap-2 mb-1">
        <span class="spinner-border spinner-border-sm text-primary flex-shrink-0"></span>
        <span class="fw-semibold small">Analyzing recordings…</span>
        <span class="ms-auto small text-muted">{{ analysisJobCount }} job{{ analysisJobCount === 1 ? "" : "s" }} active · {{ analysisProgress }}% avg</span>
      </div>
      <div class="progress" style="height: 4px;">
        <div class="progress-bar progress-bar-striped progress-bar-animated" :style="{ width: analysisProgress + '%' }"></div>
      </div>
    </div>

    <!-- ─── Search Tab ──────────────────────────────────────────────────────── -->
    <div v-show="sim.tab === 'search'">
      <div class="row g-4">
        <!-- Controls -->
        <div class="col-12 col-xl-4 col-xxl-3">
          <div class="panel">
            <!-- Source toggle -->
            <div class="panel-section">
              <div class="section-label">Query source</div>
              <div class="btn-group w-100">
                <input type="radio" class="btn-check" id="srcUpload" value="upload" v-model="sim.searchSource" />
                <label class="btn btn-sm btn-outline-primary" for="srcUpload">
                  <i class="bi bi-upload me-1"></i>Upload
                </label>
                <input type="radio" class="btn-check" id="srcLibrary" value="library" v-model="sim.searchSource" />
                <label class="btn btn-sm btn-outline-primary" for="srcLibrary">
                  <i class="bi bi-film me-1"></i>From library
                </label>
              </div>
            </div>

            <!-- Upload zone -->
            <div v-if="sim.searchSource === 'upload'" class="panel-section">
              <div
                class="drop-zone"
                :class="{ 'drop-zone--filled': !!sim.previewDataUrl }"
                @dragover.prevent="dragging = true"
                @dragleave.prevent="dragging = false"
                @drop.prevent="onDrop"
                :style="dragging ? 'border-color: var(--bs-primary)' : ''"
              >
                <template v-if="!sim.previewDataUrl">
                  <i class="bi bi-cloud-arrow-up drop-zone-icon"></i>
                  <div class="drop-zone-text">Drag an image here</div>
                  <div class="drop-zone-sub">or</div>
                  <label class="btn btn-sm btn-outline-secondary mt-1" style="cursor: pointer">
                    Browse
                    <input type="file" class="d-none" accept="image/*" @change="onFileChange" />
                  </label>
                </template>
                <template v-else>
                  <img :src="sim.previewDataUrl" class="drop-zone-preview" alt="Query image" />
                  <button class="drop-zone-clear" type="button" @click="clearFile" title="Remove">
                    <i class="bi bi-x-circle-fill"></i>
                  </button>
                </template>
              </div>
            </div>

            <!-- Library picker -->
            <div v-else class="panel-section">
              <input
                v-model="libraryFilter"
                type="text"
                class="form-control form-control-sm mb-2"
                placeholder="Filter by name or channel…"
              />
              <div class="library-list">
                <div v-if="loadingLibrary" class="library-state">
                  <span class="spinner-border spinner-border-sm me-2 text-primary"></span>
                  <span class="text-muted small">Loading…</span>
                </div>
                <div v-else-if="filteredLibrary.length === 0" class="library-state text-muted small">
                  No recordings found.
                </div>
                <button
                  v-for="rec in filteredLibrary"
                  :key="rec.recordingId"
                  type="button"
                  class="library-item"
                  :class="{ 'library-item--selected': sim.selectedRecording?.recordingId === rec.recordingId }"
                  @click="sim.selectedRecording = rec"
                >
                  <img :src="`${fileUrl}/${videoCover(rec)}`" class="library-thumb" alt="" />
                  <div class="library-meta">
                    <div class="library-channel">{{ rec.channelName }}</div>
                    <div class="library-filename">{{ rec.filename }}</div>
                  </div>
                  <i v-if="sim.selectedRecording?.recordingId === rec.recordingId" class="bi bi-check2-circle text-white ms-auto flex-shrink-0"></i>
                </button>
              </div>
            </div>

            <!-- Threshold slider -->
            <div class="panel-section">
              <div class="d-flex justify-content-between align-items-baseline mb-1">
                <div class="section-label mb-0">Similarity threshold</div>
                <div class="slider-value" :style="{ color: simColor(sim.searchSimilarity) }">
                  {{ Math.round(sim.searchSimilarity * 100) }}%
                </div>
              </div>
              <input
                type="range" class="form-range" min="0" max="1" step="0.01"
                v-model.number="sim.searchSimilarity"
              />
              <div class="slider-labels"><span>Any</span><span>Exact</span></div>
            </div>

            <!-- Limit -->
            <div class="panel-section">
              <div class="section-label">Max results</div>
              <input type="number" class="form-control form-control-sm" min="1" max="200" v-model.number="sim.searchLimit" />
            </div>

            <!-- Action -->
            <div class="panel-section">
              <button class="btn btn-primary w-100" :disabled="searching || !canSearch" @click="doSearch">
                <span v-if="searching" class="spinner-border spinner-border-sm me-2"></span>
                <i v-else class="bi bi-search me-2"></i>
                {{ searching ? "Searching…" : "Search" }}
              </button>
              <div v-if="searchError" class="error-banner mt-2">
                <i class="bi bi-exclamation-triangle-fill me-1"></i>{{ searchError }}
              </div>
            </div>
          </div>
        </div>

        <!-- Results -->
        <div class="col-12 col-xl-8 col-xxl-9">
          <div v-if="!sim.searchResults && !searching" class="empty-state">
            <i class="bi bi-images empty-icon"></i>
            <div class="empty-title">Ready to search</div>
            <div class="empty-sub">Upload an image or pick a recording from the library, then hit Search.</div>
          </div>

          <div v-else-if="searching" class="empty-state">
            <div class="spinner-border text-primary empty-spinner"></div>
            <div class="empty-title mt-3">Searching…</div>
            <div class="empty-sub">Comparing feature vectors across your library.</div>
          </div>

          <template v-else-if="sim.searchResults">
            <div class="results-bar mb-3">
              <div class="results-count">
                <span class="results-num">{{ sim.searchResults.results?.length ?? 0 }}</span>
                <span class="results-word">results</span>
              </div>
              <span class="meta-pill">threshold {{ Math.round((sim.searchResults.similarityThreshold ?? 0) * 100) }}%</span>
            </div>

            <div v-if="sim.searchResults.results?.length === 0" class="empty-state">
              <i class="bi bi-slash-circle empty-icon"></i>
              <div class="empty-title">No matches</div>
              <div class="empty-sub">Try lowering the similarity threshold.</div>
            </div>

            <div v-else class="row g-3">
              <div
                v-for="match in sim.searchResults.results"
                :key="match.recording?.recordingId"
                class="col-12 col-sm-6 col-xl-4"
              >
                <RouterLink :to="`/recordings/${match.recording?.recordingId}`" class="result-card">
                  <div class="result-thumb-wrap">
                    <img :src="`${fileUrl}/${videoCover(match.recording!)}`" class="result-thumb" alt="" />
                    <div class="result-badges">
                      <span class="sim-badge" :class="`sim-badge--${simTier(match.similarity ?? 0)}`">
                        {{ Math.round((match.similarity ?? 0) * 100) }}%
                      </span>
                      <span v-if="match.bestTimestamp" class="ts-badge">
                        <i class="bi bi-clock"></i> {{ formatTime(match.bestTimestamp) }}
                      </span>
                    </div>
                  </div>
                  <div class="result-meta">
                    <div class="result-channel">{{ match.recording?.channelName }}</div>
                    <div class="result-filename">{{ match.recording?.filename }}</div>
                  </div>
                </RouterLink>
              </div>
            </div>
          </template>
        </div>
      </div>

    </div>

    <!-- ─── Group Tab ──────────────────────────────────────────────────────── -->
    <div v-show="sim.tab === 'group'">
      <div class="row g-4">
      <!-- Controls -->
      <div class="col-12 col-xl-4 col-xxl-3">
        <div class="panel">
          <!-- Threshold slider -->
          <div class="panel-section">
            <div class="d-flex justify-content-between align-items-baseline mb-1">
              <div class="section-label mb-0">Similarity threshold</div>
              <div class="slider-value" :style="{ color: simColor(sim.groupSimilarity) }">
                {{ Math.round(sim.groupSimilarity * 100) }}%
              </div>
            </div>
            <input
              type="range" class="form-range" min="0" max="1" step="0.01"
              v-model.number="sim.groupSimilarity"
            />
            <div class="slider-labels"><span>Any</span><span>Exact</span></div>
          </div>

          <!-- Pair limit -->
          <div class="panel-section">
            <div class="section-label">Pair comparison limit</div>
            <input type="number" class="form-control form-control-sm" min="1" max="100000" v-model.number="sim.pairLimit" />
            <div class="hint-text">Lower for faster results on large libraries.</div>
          </div>

          <!-- Singletons -->
          <div class="panel-section">
            <div class="d-flex justify-content-between align-items-center">
              <div>
                <div class="section-label mb-0">Include singletons</div>
                <div class="hint-text mt-0">Recordings with no similar neighbour.</div>
              </div>
              <div class="form-check form-switch ms-3 mb-0">
                <input class="form-check-input" type="checkbox" role="switch" id="singletons" v-model="sim.includeSingletons" />
                <label class="form-check-label" for="singletons"></label>
              </div>
            </div>
          </div>

          <!-- Action -->
          <div class="panel-section">
            <button class="btn btn-primary w-100" :disabled="grouping" @click="doGroup">
              <span v-if="grouping" class="spinner-border spinner-border-sm me-2"></span>
              <i v-else class="bi bi-diagram-3 me-2"></i>
              {{ grouping ? "Grouping…" : "Find Groups" }}
            </button>
            <div v-if="groupError" class="error-banner mt-2">
              <i class="bi bi-exclamation-triangle-fill me-1"></i>{{ groupError }}
            </div>
          </div>
        </div>
      </div>

      <!-- Group results -->
      <div class="col-12 col-xl-8 col-xxl-9">
        <div v-if="!sim.groupResults && !grouping" class="empty-state">
          <i class="bi bi-collection empty-icon"></i>
          <div class="empty-title">No groups yet</div>
          <div class="empty-sub">Set a threshold and click Find Groups to cluster your library.</div>
        </div>

        <div v-else-if="grouping" class="empty-state">
          <div class="spinner-border text-primary empty-spinner"></div>
          <div class="empty-title mt-3">Building groups…</div>
          <div class="empty-sub">Clustering recordings by visual similarity.</div>
        </div>

        <template v-else-if="sim.groupResults">
          <div class="results-bar mb-3">
            <div class="results-count">
              <span class="results-num">{{ sim.groupResults.groupCount }}</span>
              <span class="results-word">groups</span>
            </div>
            <span class="meta-pill">threshold {{ Math.round((sim.groupResults.similarityThreshold ?? 0) * 100) }}%</span>
          </div>

          <div v-if="sim.groupResults.groupCount === 0" class="empty-state">
            <i class="bi bi-slash-circle empty-icon"></i>
            <template v-if="(sim.groupResults.analyzedCount ?? 0) === 0">
              <div class="empty-title">No analyzed recordings</div>
              <div class="empty-sub mb-3">
                No recordings have feature vectors yet. Analyze all recordings to enable grouping.
              </div>
              <button class="btn btn-primary btn-sm" :disabled="analyzingAll" @click="analyzeAll">
                <span v-if="analyzingAll" class="spinner-border spinner-border-sm me-1"></span>
                <i v-else class="bi bi-cpu me-1"></i>
                {{ analyzingAll ? "Enqueuing…" : "Analyze All Recordings" }}
              </button>
              <div v-if="analyzeAllMessage" class="mt-2 small text-success">{{ analyzeAllMessage }}</div>
            </template>
            <template v-else>
              <div class="empty-title">No groups found</div>
              <div class="empty-sub">
                {{ sim.groupResults.analyzedCount }} recording{{ sim.groupResults.analyzedCount === 1 ? '' : 's' }} analyzed.
                Try lowering the similarity threshold.
              </div>
            </template>
          </div>

          <div v-else class="group-list">
            <div v-for="group in sim.groupResults.groups" :key="group.groupId" class="group-card">
              <!-- Header -->
              <div class="group-header">
                <div class="d-flex align-items-center gap-2 flex-wrap">
                  <span class="group-count">{{ group.videos?.length }}</span>
                  <span class="group-label">Group {{ group.groupId }}</span>
                  <span class="sim-pill" :class="`sim-pill--${simTier(group.maxSimilarity ?? 0)}`">
                    ≤ {{ Math.round((group.maxSimilarity ?? 0) * 100) }}% similar
                  </span>
                </div>
              </div>

              <!-- Videos -->
              <div class="row g-3 p-3">
                <div v-for="vid in group.videos" :key="vid.recordingId" class="col-12 col-sm-6 col-xl-4">
                  <RouterLink :to="`/recordings/${vid.recordingId}`" class="result-card">
                    <div class="result-thumb-wrap">
                      <img :src="`${fileUrl}/${videoCover(vid)}`" alt="" class="result-thumb" />
                    </div>
                    <div class="result-meta">
                      <div class="result-channel">{{ vid.channelName }}</div>
                      <div class="result-filename">{{ vid.filename }}</div>
                    </div>
                  </RouterLink>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { createClient } from "@/services/api/v1/ClientFactory";
import type { DbRecording } from "@/services/api/v1/MediaSinkClient";
import { useJobStore } from "@/stores/job";
import { useSimilarityStore } from "@/stores/similarity";
import { videoCover } from "@/utils/video";

const fileUrl = inject("fileUrl") as string;
const jobStore = useJobStore();
const sim = useSimilarityStore();

// ─── Analysis progress ─────────────────────────────────────────────────────────
const analysisJobs = computed(() =>
  jobStore.all.filter((j) => j.task === "analyze-frames" && j.active),
);
const analysisJobCount = computed(() => analysisJobs.value.length);
const analysisProgress = computed(() => {
  if (analysisJobCount.value === 0) return 0;
  const avg = analysisJobs.value.reduce((sum, j) => sum + parseFloat(j.progress ?? "0"), 0) / analysisJobCount.value;
  return Math.round(avg);
});

// ─── Shared library ────────────────────────────────────────────────────────────
const library = ref<DbRecording[]>([]);
const loadingLibrary = ref(false);

onMounted(async () => {
  loadingLibrary.value = true;
  try {
    library.value = (await createClient().videos.videosList()) ?? [];
  } catch {
    // non-fatal
  } finally {
    loadingLibrary.value = false;
  }
});

// ─── Search ────────────────────────────────────────────────────────────────────
// uploadFile is ephemeral (can't serialize a File); previewDataUrl survives navigation
const uploadFile = ref<File | null>(null);
const dragging = ref(false);
const libraryFilter = ref("");
const searching = ref(false);
const searchError = ref<string | null>(null);

const filteredLibrary = computed(() => {
  const q = libraryFilter.value.toLowerCase();
  if (!q) return library.value;
  return library.value.filter(
    (r) => r.filename.toLowerCase().includes(q) || r.channelName.toLowerCase().includes(q),
  );
});

const canSearch = computed(() =>
  sim.searchSource === "upload" ? !!(uploadFile.value || sim.previewDataUrl) : !!sim.selectedRecording,
);

const fileToDataUrl = (file: File): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });

const setFile = async (file: File) => {
  uploadFile.value = file;
  sim.previewDataUrl = await fileToDataUrl(file);
};

const onFileChange = (e: Event) => {
  const f = (e.target as HTMLInputElement).files?.[0];
  if (f) setFile(f);
};

const onDrop = (e: DragEvent) => {
  dragging.value = false;
  const f = e.dataTransfer?.files?.[0];
  if (f && f.type.startsWith("image/")) setFile(f);
};

const clearFile = () => {
  uploadFile.value = null;
  sim.previewDataUrl = null;
  sim.searchResults = null;
};

const dataUrlToFile = async (dataUrl: string): Promise<File> => {
  const res = await fetch(dataUrl);
  const blob = await res.blob();
  return new File([blob], "query.jpg", { type: blob.type || "image/jpeg" });
};

const doSearch = async () => {
  searching.value = true;
  sim.searchResults = null;
  searchError.value = null;
  try {
    let file: File;
    if (sim.searchSource === "upload") {
      file = uploadFile.value ?? (await dataUrlToFile(sim.previewDataUrl!));
    } else {
      const imgUrl = `${fileUrl}/${videoCover(sim.selectedRecording!)}`;
      const resp = await fetch(imgUrl);
      if (!resp.ok) throw new Error(`Could not fetch preview frame (${resp.status})`);
      const blob = await resp.blob();
      file = new File([blob], "query.jpg", { type: blob.type || "image/jpeg" });
    }
    sim.searchResults = await createClient().analysis.searchImageCreate({
      file,
      similarity: sim.searchSimilarity,
      limit: sim.searchLimit,
    });
  } catch (e) {
    searchError.value = e instanceof Error ? e.message : "Search failed.";
  } finally {
    searching.value = false;
  }
};

// ─── Group ─────────────────────────────────────────────────────────────────────
const grouping = ref(false);
const groupError = ref<string | null>(null);

const doGroup = async () => {
  grouping.value = true;
  sim.groupResults = null;
  groupError.value = null;
  try {
    sim.groupResults = await createClient().analysis.groupCreate({
      similarity: sim.groupSimilarity,
      pairLimit: sim.pairLimit,
      includeSingletons: sim.includeSingletons,
    });
  } catch (e) {
    groupError.value = e instanceof Error ? e.message : "Grouping failed.";
  } finally {
    grouping.value = false;
  }
};

// ─── Analyze all ───────────────────────────────────────────────────────────────
const analyzingAll = ref(false);
const analyzeAllMessage = ref<string | null>(null);

const analyzeAll = async () => {
  analyzingAll.value = true;
  analyzeAllMessage.value = null;
  try {
    const result = (await createClient().analysis.allCreate()) as Record<string, number>;
    const count = result["enqueued"] ?? 0;
    analyzeAllMessage.value = `${count} job${count === 1 ? "" : "s"} enqueued.`;
  } catch (e) {
    analyzeAllMessage.value = e instanceof Error ? e.message : "Failed to enqueue jobs.";
  } finally {
    analyzingAll.value = false;
  }
};

// ─── Helpers ───────────────────────────────────────────────────────────────────
const simTier = (v: number) => (v >= 0.9 ? "high" : v >= 0.75 ? "mid" : v >= 0.55 ? "low" : "none");

const simColor = (v: number) => {
  if (v >= 0.9) return "#198754";
  if (v >= 0.75) return "#0dcaf0";
  if (v >= 0.55) return "#ffc107";
  return "#6c757d";
};

const formatTime = (s: number) => `${Math.floor(s / 60)}:${String(Math.floor(s % 60)).padStart(2, "0")}`;
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bs;

$primary: #301240;
$radius: 0.5rem;
$panel-bg: var(--bs-body-bg);
$section-gap: 0.85rem;

// ─── Analysis progress banner ─────────────────────────────────────────────────
.analysis-progress-bar {
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: $radius;
  padding: 0.65rem 0.9rem;
}

// ─── Control panel ────────────────────────────────────────────────────────────
.panel {
  background: $panel-bg;
  border: 1px solid var(--bs-border-color);
  border-radius: $radius;
  overflow: hidden;
}

.panel-section {
  padding: $section-gap;
  border-bottom: 1px solid var(--bs-border-color);

  &:last-child {
    border-bottom: none;
  }
}

.section-label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--bs-secondary-color);
  margin-bottom: 0.6rem;
}

.slider-value {
  font-size: 1.4rem;
  font-weight: 800;
  line-height: 1;
  transition: color 0.2s;
}

.slider-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--bs-secondary-color);
  margin-top: 0.1rem;
}

.hint-text {
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  margin-top: 0.35rem;
}

.error-banner {
  background: rgba(#cc2255, 0.1);
  border: 1px solid rgba(#cc2255, 0.3);
  border-radius: 0.35rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.78rem;
  color: #cc2255;
}

// ─── Drop zone ────────────────────────────────────────────────────────────────
.drop-zone {
  border: 2px dashed var(--bs-border-color);
  border-radius: 0.4rem;
  min-height: 140px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 1.25rem;
  text-align: center;
  transition: border-color 0.15s, background 0.15s;
  background: var(--bs-tertiary-bg);
  cursor: pointer;

  &:hover {
    border-color: $primary;
    background: rgba($primary, 0.04);
  }

  &--filled {
    border-style: solid;
    border-color: var(--bs-border-color);
    padding: 0;
    overflow: hidden;
    cursor: default;
    position: relative;
  }
}

.drop-zone-icon {
  font-size: 2rem;
  color: var(--bs-secondary-color);
  margin-bottom: 0.5rem;
}

.drop-zone-text {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--bs-body-color);
}

.drop-zone-sub {
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
}

.drop-zone-preview {
  width: 100%;
  height: 130px;
  object-fit: cover;
  display: block;
}

.drop-zone-clear {
  position: absolute;
  top: 6px;
  right: 6px;
  background: rgba(black, 0.5);
  border: none;
  color: white;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: rgba(black, 0.75);
  }
}

// ─── Library picker ───────────────────────────────────────────────────────────
.library-list {
  border: 1px solid var(--bs-border-color);
  border-radius: 0.35rem;
  max-height: 220px;
  overflow-y: auto;
  overflow-x: hidden;
}

.library-state {
  padding: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.library-item {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 0.45rem 0.6rem;
  border: none;
  border-bottom: 1px solid var(--bs-border-color);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.1s;
  gap: 0.6rem;

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    background: var(--bs-tertiary-bg);
  }

  &--selected {
    background: $primary !important;
    color: white;

    .library-channel,
    .library-filename {
      color: rgba(white, 0.85) !important;
    }
  }
}

.library-thumb {
  width: 52px;
  height: 30px;
  object-fit: cover;
  border-radius: 3px;
  flex-shrink: 0;
}

.library-meta {
  min-width: 0;
  flex: 1;
}

.library-channel {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--bs-secondary-color);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.library-filename {
  font-size: 0.78rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

// ─── Results bar ─────────────────────────────────────────────────────────────
.results-bar {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
}

.results-count {
  display: flex;
  align-items: baseline;
  gap: 0.35rem;
}

.results-num {
  font-size: 1.5rem;
  font-weight: 800;
  color: $primary;
  line-height: 1;
}

.results-word {
  font-size: 0.85rem;
  color: var(--bs-secondary-color);
}

.meta-pill {
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: 20px;
  padding: 0.15rem 0.65rem;
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
}

// ─── Result cards ─────────────────────────────────────────────────────────────
.result-card {
  display: block;
  text-decoration: none;
  border-radius: $radius;
  overflow: hidden;
  border: 1px solid var(--bs-border-color);
  background: $panel-bg;
  transition: transform 0.15s, box-shadow 0.15s;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(black, 0.12);
  }
}

.result-thumb-wrap {
  position: relative;
  aspect-ratio: 16/9;
  overflow: hidden;
  background: #111;
}

.result-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.result-badges {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0.5rem;
  background: linear-gradient(transparent, rgba(black, 0.7));
  display: flex;
  align-items: flex-end;
  gap: 0.35rem;
}

.sim-badge {
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.2rem 0.5rem;
  border-radius: 20px;

  &--high { background: #198754; color: white; }
  &--mid  { background: #0dcaf0; color: #000; }
  &--low  { background: #ffc107; color: #000; }
  &--none { background: rgba(white, 0.2); color: white; }
}

.ts-badge {
  font-size: 0.7rem;
  color: rgba(white, 0.8);
  display: flex;
  align-items: center;
  gap: 0.2rem;
}

.result-meta {
  padding: 0.5rem 0.6rem 0.6rem;
}

.result-channel {
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--bs-secondary-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.result-filename {
  font-size: 0.78rem;
  color: var(--bs-body-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

// ─── Empty states ─────────────────────────────────────────────────────────────
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 240px;
  text-align: center;
  color: var(--bs-secondary-color);
}

.empty-icon {
  font-size: 3.5rem;
  opacity: 0.2;
  margin-bottom: 0.75rem;
}

.empty-spinner {
  width: 3rem;
  height: 3rem;
}

.empty-title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--bs-body-color);
}

.empty-sub {
  font-size: 0.82rem;
  margin-top: 0.25rem;
  max-width: 320px;
}

// ─── Group cards ──────────────────────────────────────────────────────────────
.group-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.group-card {
  border: 1px solid var(--bs-border-color);
  border-radius: $radius;
  overflow: hidden;
  background: $panel-bg;
}

.group-header {
  display: flex;
  align-items: center;
  padding: 0.65rem 1rem;
  background: var(--bs-tertiary-bg);
  border-bottom: 1px solid var(--bs-border-color);
  gap: 0.5rem;
}

.group-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 26px;
  height: 26px;
  padding: 0 6px;
  background: $primary;
  color: white;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 700;
}

.group-label {
  font-size: 0.85rem;
  font-weight: 600;
}

.sim-pill {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.15rem 0.55rem;
  border-radius: 20px;

  &--high { background: rgba(#198754, 0.15); color: #198754; border: 1px solid rgba(#198754, 0.3); }
  &--mid  { background: rgba(#0dcaf0, 0.15); color: #0a9cbf; border: 1px solid rgba(#0dcaf0, 0.3); }
  &--low  { background: rgba(#ffc107, 0.15); color: #997404; border: 1px solid rgba(#ffc107, 0.3); }
  &--none { background: var(--bs-tertiary-bg); color: var(--bs-secondary-color); border: 1px solid var(--bs-border-color); }
}




</style>
