<template>
  <ChannelModal @save="save" @close="showModal = false" title="Edit Stream" :saving="saving" :is-paused="isPaused" :channel-disabled="true" :clear="false" :channel-id="channelId" :show="showModal" :channel-name="channelName" :display-name="displayName" :url="url" :min-duration="minDuration" :skip-start="skipStart" />

  <!-- Search bar -->
  <div class="row">
    <div class="col">
      <div class="input-group mb-2 align-middle">
        <input ref="searchField" autocapitalize="off" autocomplete="off" class="form-control border-secondary" type="text" name="search" placeholder="search ... #tag" v-model="searchVal" />
        <span class="input-group-text bg-danger-subtle" v-if="searchVal != ''">
          <i class="bi bi-x-lg text-danger fs-4" @click="searchVal = ''" />
        </span>
        <span class="input-group-text" style="color: deeppink">
          <i v-if="favs" class="bi bi-heart-fill fs-4" @click="favs = false" />
          <i v-else class="bi bi-heart fs-4" @click="favs = true" />
        </span>
      </div>
    </div>
  </div>
  <!-- Search bar -->

  <!-- Body -->
  <LoadIndicator :busy="isLoading">
    <div class="row">
      <!-- Search -->
      <div v-if="searchVal !== '' || favs" class="col">
        <div class="row">
          <FillNotice v-if="searchResults.length === 0">
            <h1>No results</h1>
          </FillNotice>
          <div v-else v-for="channel in searchResults" :key="channel.channelId" :class="channelItemClass">
            <ChannelItem :channel="channel" />
          </div>
        </div>
      </div>
      <!-- No search -->
      <div v-else class="col">
        <ul class="nav nav-tabs border-primary" id="myTab" role="tablist">
          <li class="nav-item" role="presentation">
            <button class="nav-link d-flex justify-content-between" @click="() => show('live')" :class="{ active: tab === 'live' }" id="home-tab" data-bs-toggle="tab" data-bs-target="#home" type="button" role="tab" aria-controls="home" aria-selected="true">
              <span class="d-none d-lg-inline">Recording</span>
              <span class="d-flex justify-content-between">
                <span class="d-lg-none">Rec</span>
                <span class="recording-number">{{ channelStore.recordingStreams.length }}</span>
              </span>
            </button>
          </li>
          <li class="nav-item" role="presentation">
            <button class="nav-link d-flex justify-content-between" @click="() => show('offline')" :class="{ active: tab === 'offline' }" id="profile-tab" data-bs-toggle="tab" data-bs-target="#profile" type="button" role="tab" aria-controls="profile" aria-selected="false">
              <span class="d-none d-lg-inline">Offline</span>
              <span class="d-flex justify-content-between">
                <span class="d-lg-none">Off</span>
                <span class="recording-number">{{ channelStore.notRecordingStreams.length }}</span>
              </span>
            </button>
          </li>
          <li class="nav-item" role="presentation">
            <button class="nav-link d-flex justify-content-between" @click="() => show('disabled')" :class="{ active: tab === 'disabled' }" id="disabled-tab" data-bs-toggle="tab" data-bs-target="#disabled" type="button" role="tab" aria-controls="disabled" aria-selected="false">
              <span class="d-none d-lg-inline">Disabled</span>
              <span class="d-flex justify-content-between">
                <span class="d-lg-none">Disabled</span>
                <span class="recording-number">{{ channelStore.disabledStreams.length }}</span>
              </span>
            </button>
          </li>
        </ul>

        <div class="tab-content py-2" id="myTabContent">
          <div class="tab-pane fade" :class="{ 'active show': tab === 'live' }" id="home" role="tabpanel" aria-labelledby="home-tab">
            <div class="row">
              <div v-for="channel in channelStore.recordingStreams" :key="channel.channelId" :class="channelItemClass">
                <ChannelItem :channel="channel" @edit="editChannel" />
              </div>
              <FillNotice v-if="channelStore.recordingStreams.length === 0">
                <h1>No recording streams</h1>
              </FillNotice>
            </div>
          </div>

          <div class="tab-pane fade" :class="{ 'active show': tab === 'offline' }" id="profile" role="tabpanel" aria-labelledby="profile-tab">
            <div class="row">
              <div v-for="channel in channelStore.notRecordingStreams" :key="channel.channelId" :class="channelItemClass">
                <ChannelItem :channel="channel" @edit="editChannel" />
              </div>
              <FillNotice v-if="channelStore.notRecordingStreams.length === 0">
                <h1>No offline streams</h1>
              </FillNotice>
            </div>
          </div>

          <div class="tab-pane fade" :class="{ 'active show': tab === 'disabled' }" id="disabled" role="tabpanel" aria-labelledby="disabled-tab">
            <div class="row">
              <div v-for="channel in channelStore.disabledStreams" :key="channel.channelId" :class="channelItemClass">
                <ChannelItem :channel="channel" @edit="editChannel" />
              </div>
              <FillNotice v-if="channelStore.disabledStreams.length === 0">
                <h1>No disabled streams</h1>
              </FillNotice>
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- Body end -->
  </LoadIndicator>
</template>

<script setup lang="ts">
import type { DatabaseChannel as ChannelResponse } from "@/services/api/v1/MediaSinkClient";
import ChannelItem from "@/components/ChannelItem.vue";
import ChannelModal from "@/components/modals/ChannelModal.vue";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import LoadIndicator from "@/components/LoadIndicator.vue";
import { sortChannel, useChannelStore } from "@/stores/channel.ts";
import type { ChannelUpdate } from "@/types/channel";
import FillNotice from "@/components/FillNotice.vue";

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const route = useRoute();
const channelItemClass = "col-lg-6 col-xl-4 col-xxl-3 col-md-6 col-sm-8";
const channelId = ref<number>();
const showModal = ref(false);
const channelName = ref("");
const displayName = ref("");
const isPaused = ref(false);
const url = ref("");
const minDuration = ref(20);
const skipStart = ref(0);
const favs = ref(route.query.fav === "1");
const searchField = ref<HTMLInputElement | null>(null);

const isLoading = ref(true);

const saving = ref(false);

const searchVal = ref<string>((route.query.search || route.query.tag || route.params.tag || "") as string);
//const tagFilter = ref<string>((route.params.tag || route.query.tag || "") as string);

const router = useRouter();

const tab = ref(route.params.tab);

const channelStore = useChannelStore();

// --------------------------------------------------------------------------------------
// Computes
// --------------------------------------------------------------------------------------

const search = computed(() => searchVal.value.trim().toLowerCase());

const searchTerms = computed(() =>
  search.value
    .split(" ")
    .filter((s) => s[0] !== "#")
    .join(" "),
);

const tagTerms = computed(() =>
  search.value
    .split(" ")
    .filter((s) => s[0] === "#")
    .map((s) => s.slice(1, s.length))
    .join(" "),
);

const searchResults = computed(() =>
  channelStore.all
    .filter((channel) => searchFilter(channel, searchTerms.value, tagTerms.value))
    .filter((channel) => (favs.value ? channel.fav : true)) // Only use, if defined.
    .sort(sortChannel),
);

// --------------------------------------------------------------------------------------
// Functions
// --------------------------------------------------------------------------------------

const searchFilter = (channel: ChannelResponse, search: string, tag: string): boolean => {
  return (search && search.length > 0 ? channel.channelName!.indexOf(search) !== -1 : true) && (tag && tag.length > 0 && channel.tags ? channel.tags.some((t) => t === tag) : true);
};

const save = async (channel: ChannelUpdate) => {
  try {
    saving.value = true;
    await channelStore.save(channel.channelId, channel);
    showModal.value = false;
  } catch (e) {
    alert(e);
  } finally {
    saving.value = false;
  }
};

const editChannel = (channel: ChannelResponse) => {
  channelId.value = channel.channelId!;
  channelName.value = channel.channelName;
  displayName.value = channel.displayName;
  isPaused.value = channel.isPaused;
  url.value = channel.url;
  skipStart.value = channel.skipStart;
  minDuration.value = channel.minDuration;
  showModal.value = true;
};

const show = (tabName: string) => {
  router.push({ params: { tab: tabName } });
};

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(favs, (val) => router.push({ query: { fav: val ? "1" : "0" } }));

watch(searchVal, (search) => {
  // Using router.replace forces a re-render, that's why replaceState is used.
  let params = `search=${encodeURIComponent(search)}`;
  if (route.query.fav) {
    params += `&fav=1`;
  }
  window.history.replaceState({}, "", `${window.location.pathname}?${params}`);
});

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onMounted(async () => {
  await channelStore.load();
  isLoading.value = false;
});
</script>

<style lang="scss" scoped>
.nav-item .active {
  background-color: #4b4e6d;
  color: white;
  border: 1px solid #4b4e6d;
}

.recording-number {
  font-size: 0.8em;
}
</style>
