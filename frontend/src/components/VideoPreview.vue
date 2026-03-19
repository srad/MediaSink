<template>
  <div class="position-relative preview-container rounded-top-2">
    <span class="recording-indicator position-absolute">
      <i class="bi fs-4 text-danger blink bi-record-fill pulse" v-if="props.isRecording"></i>
    </span>

    <div class="position-relative w-100 h-100" @mouseenter="paused = false" @mouseleave="paused = true">
      <img
        v-if="showStaticPreview"
        class="w-100 h-100 object-fit-cover rounded-top-2"
        alt="preview"
        loading="lazy"
        :src="props.previewImage" />
      <ImageFramePlayer
        v-else
        class="w-100 h-100 rounded-top-2"
        :frames="props.previewFrames!"
        :pause="paused"
        :duration="1"
        :fps="4" />
    </div>

    <div v-if="isRecording" class="recording-time-overlay bg-danger position-absolute badge rounded-2 opacity-75 text-white">{{ minutes }}:{{ seconds }}min</div>
    <div class="channel-info-overlay bg-dark position-absolute badge rounded-2 opacity-75 text-white" v-if="props.recordingsSize && props.recordingsCount">{{ (props.recordingsSize / 1024 / 1024 / 1024).toFixed(1) }}GB ({{ props.recordingsCount }})</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import ImageFramePlayer from "../components/ImageFramePlayer.vue";

const thread = ref<null | ReturnType<typeof setTimeout>>(null);

const props = defineProps<{
  minRecording?: number;
  recordingsSize?: number;
  recordingsCount?: number;
  isRecording?: boolean;
  data: string | number;
  previewImage: string;
  previewFrames?: string[];
}>();

const secRecording = ref((props.minRecording || 0) * 60);

const paused = ref(true);
const showStaticPreview = computed(() => !props.previewFrames?.length || paused.value);

const minutes = computed(() => (secRecording.value / 60).toFixed(0));
const seconds = computed(() => {
  const x = (secRecording.value % 60).toFixed(0);
  return x.length < 2 ? "0" + String(x) : x;
});

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onMounted(() => {
  if (props.isRecording) {
    // Increase the seconds to indicate liveness.
    thread.value = setInterval(() => {
      secRecording.value += 1;
    }, 1000);
  }
});

onUnmounted(() => {
  if (props.isRecording && thread.value) {
    clearInterval(thread.value);
  }
});
</script>

<style scoped>
.preview-container {
  min-height: 100px;
}

.preview-container img,
.preview-container video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  vertical-align: middle;
  z-index: -1;
}

.recording-indicator {
  top: 1%;
  left: 5%;
}

.channel-info-overlay {
  bottom: 5%;
  right: 5%;
  z-index: 1;
}

.recording-time-overlay {
  bottom: 5%;
  left: 5%;
  z-index: 1;
}
</style>
