<template>
  <div class="user-select-none">
    <ModalConfirmDialog :show="showConfirmRecording" @cancel="showConfirmRecording = false" @confirm="record">
      <template v-slot:header>
        <span v-if="isRecording">Stop Recording</span>
        <span v-else>Resume Recording</span>
      </template>
      <template v-slot:body>
        <div v-if="isRecording">Do you want to stop recording?</div>
        <div v-else>Do you want to continue recording?</div>
      </template>
    </ModalConfirmDialog>

    <nav class="navbar navbar-dark navbar-expand-lg fixed-top d-flex bg-primary shadow-sm" style="opacity: 0.9">
      <div class="container-fluid">
        <div class="d-flex align-items-center">
          <AppBrand class="mr-auto" :title="title" />
          <HeartBeat />
        </div>

        <div class="offcanvas offcanvas-end bg-dark" :class="{ show: showNav }" data-bs-backdrop="static" tabindex="-1" aria-labelledby="collapsibleNavbarLabel" id="collapsibleNavbar">
          <div class="offcanvas-header bg-primary text-white">
            <AppBrand class="mr-auto" :title="title" />
            <button type="button" class="btn-close btn-close-white" @click="showNav = !showNav"></button>
          </div>
          <div class="offcanvas-body">
            <ul class="navbar-nav flex-grow-1">
              <li class="nav-item" style="cursor: pointer" v-for="link in props.routes" :key="link.url">
                <RouterLink :to="link.url" class="nav-link d-flex gap-2" @click="showNav = false">
                  <i class="bi" :class="link.icon"></i>
                  <span data-bs-dismiss="offcanvas" data-bs-target="#collapsibleNavbar">{{ link.title }}</span>
                </RouterLink>
              </li>
              <li class="dropdown-item d-inline d-lg-none">
                <RouterLink to="/admin" class="nav-link d-flex gap-2 bg-transparent" @click="showNav = false">
                  <i class="bi bi-sliders"></i>
                  <span>Admin</span>
                </RouterLink>
              </li>
              <li class="dropdown-item d-inline d-lg-none">
                <RouterLink to="/processes" class="nav-link d-flex gap-2 bg-transparent" @click="showNav = false">
                  <i class="bi bi-cpu-fill"></i>
                  <span>Processes</span>
                </RouterLink>
              </li>
              <li class="dropdown-item d-inline d-lg-none">
                <RouterLink to="/info" class="nav-link d-flex gap-2 bg-transparent" @click="showNav = false">
                  <i class="bi bi-info-circle-fill"></i>
                  <span>System Info</span>
                </RouterLink>
              </li>
              <li class="dropdown-item d-inline d-lg-none">
                <RouterLink to="/monitoring" class="nav-link d-flex gap-2 bg-transparent" @click="showNav = false">
                  <i class="bi bi-binoculars-fill"></i>
                  <span>Monitoring</span>
                </RouterLink>
              </li>
              <li class="dropdown-item d-inline d-lg-none">
                <a
                  href="#"
                  class="nav-link d-flex gap-2 bg-transparent"
                  @click="
                    showNav = false;
                    emit('logout');
                  ">
                  <i class="bi bi-door-open-fill"></i>
                  <span>Logout</span>
                </a>
              </li>
              <li class="dropdown-item d-inline d-lg-none">
                <a href="#" class="nav-link d-flex gap-2 bg-transparent">
                  <DarkModelToggleButton :checkbox="true" />
                </a>
              </li>
            </ul>
            <ul class="navbar-nav">
              <li class="dropdown-item d-inline d-lg-none my-2"></li>
              <li class="nav-item d-flex align-items-center">
                <DiskStatus :pcent="diskAvailablePercentage" />
              </li>
              <li class="dropdown-item d-inline d-lg-none my-2">
                <a href="#" class="nav-link d-flex gap-2 bg-transparent">Version: {{ version }}</a>
              </li>
              <li class="nav-item d-none d-lg-inline">
                <VideoControls :jobs="jobs" :total-count="jobsCount" :is-recording="isRecording" @add="emit('add')" @record="showConfirmRecording = true" />
              </li>
              <li class="nav-item d-flex align-content-center px-2 d-none d-lg-inline">
                <UserDropDown @logout="emit('logout')" />
              </li>
            </ul>
          </div>
        </div>

        <div class="d-lg-none d-flex align-items-center">
          <VideoControls :jobs="jobs" :total-count="jobsCount" :is-recording="isRecording" @add="emit('add')" @record="showConfirmRecording = true" />

          <button class="navbar-toggler d-l-none ms-2" type="button" @click="showNav = !showNav">
            <span class="navbar-toggler-icon"></span>
          </button>
        </div>
      </div>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { useSocket } from "../../composables/useSocket";
import { useChannelStore } from "../../stores/channel";
import { useJobStore } from "../../stores/job";
import { computed, inject, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import DiskStatus from "../DiskStatus.vue";
import VideoControls from "../VideoControls.vue";
import AppBrand from "../AppBrand.vue";
import ModalConfirmDialog from "../modals/ModalConfirmDialog.vue";
import type { HelpersDiskInfo } from "../../services/api/v1/MediaSinkClient";
import { createClient } from "../../services/api/v1/ClientFactory";
import { useAuthStore } from "../../stores/auth";
import UserDropDown from "../navs/UserDropDown.vue";
import DarkModelToggleButton from "../DarkModelToggleButton.vue";
import HeartBeat from "../HeartBeat.vue";

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  routes: { icon: string; url: string; title: string }[];
  title: string;
  showLogout: boolean;
}>();

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{
  (e: "add"): void;
  (e: "logout"): void;
}>();

// --------------------------------------------------------------------------------------
// Inject
// --------------------------------------------------------------------------------------
const version = inject<string>("version");

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

// Stores
const channelStore = useChannelStore();
const jobStore = useJobStore();

// Refs
const diskAvailablePercentage = ref(0);
const collapseNav = ref(true);
const isRecording = ref(false);
const route = useRoute();
const showNav = ref(false);
const showConfirmRecording = ref(false);

const authStore = useAuthStore();

let thread: undefined | ReturnType<typeof setInterval> = undefined;

const { socket: socketManager } = useSocket();

// --------------------------------------------------------------------------------------
// Computes
// --------------------------------------------------------------------------------------

const jobs = computed(() => jobStore.open);
const jobsCount = computed(() => jobStore.jobsCount);

// --------------------------------------------------------------------------------------
// Functions
// --------------------------------------------------------------------------------------

const query = async () => {
  const client = createClient();
  const [recRes, diskRes] = await Promise.all<[Promise<boolean>, Promise<HelpersDiskInfo>]>([client.isRecording(), client.info.diskList()]);
  isRecording.value = recRes;
  diskAvailablePercentage.value = diskRes.pcent;
};

const record = async () => {
  try {
    const client = createClient();
    if (isRecording.value) {
      await client.recorder.pauseCreate();
      channelStore.stop();
      isRecording.value = false;
    } else {
      await client.recorder.resumeCreate();
      isRecording.value = true;
    }
  } catch (err) {
    console.error(err);
  } finally {
    showConfirmRecording.value = false;
  }
};

const initialLoad = async () => {
  const client = createClient();
  const res = await Promise.all<[Promise<boolean>, Promise<HelpersDiskInfo>]>([client.isRecording(), client.info.diskList()]);
  const [recRes, diskRes] = res;
  diskAvailablePercentage.value = diskRes.pcent;
  isRecording.value = recRes;
};

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(route, () => (collapseNav.value = true));

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onMounted(async () => {
  if (!authStore.isLoggedIn) {
    return;
  }

  await initialLoad();
  // Poll
  thread = setInterval(query, 1000 * 10);
});

onUnmounted(() => {
  socketManager.close();
  clearInterval(thread);
});
</script>
