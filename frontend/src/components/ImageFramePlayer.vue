<template>
  <div class="image-frame-player" @click="togglePlay">
    <img :src="currentFrameSrc" alt="Video Frame" :style="imageStyles" ref="frameImage" loading="lazy" />
    <div class="opacity-50 bg-white" style="height: 10px; bottom: 0; left: 0; right: 0; background: white; position: absolute">
      <div class="h-100 bg-primary opacity-50 text-white" :style="{ width: `${progress}%` }">
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from "vue";

// ----------------------------------------------------------------------
// Props Definition
// ----------------------------------------------------------------------
interface Props {
  /** Array of image URLs (JPG, PNG, etc.) that represent the video frames. */
  frames: string[];
  /** Frames per second for playback. */
  fps: number;
  /** Boolean to control pausing the animation from the parent component. */
  pause: boolean;
  /** Duration of the overall video, used to emit time updates. */
  duration: number;
  /** Optional custom styles for the <img> tag. */
  imageStyles?: Record<string, string>;
}

const props = withDefaults(defineProps<Props>(), {
  fps: 25,
  pause: false,
  duration: 0,
  imageStyles: () => ({
    "object-fit": "contain",
    width: "100%",
    height: "100%",
    outline: "none",
  }),
});

// ----------------------------------------------------------------------
// Emits Definition
// ----------------------------------------------------------------------
const emit = defineEmits<{
  (e: "timeupdate", timecode: number): void;
  (e: "update:pause", isPaused: boolean): void;
}>();

// ----------------------------------------------------------------------
// State and Refs
// ----------------------------------------------------------------------
const currentFrameIndex = ref(0);
const isPlayingInternal = ref(true); // Internal state for animation loop
const animationInterval = ref<number | null>(null);

// Calculate the duration for each frame based on FPS
const frameDurationMs = computed(() => 1000 / props.fps);

const progress = computed(() => (currentFrameIndex.value / props.frames.length) * 100);

// ----------------------------------------------------------------------
// Computed Properties
// ----------------------------------------------------------------------

/** The source URL for the currently displayed image frame. */
const currentFrameSrc = computed(() => {
  if (props.frames.length === 0) return "";
  const index = currentFrameIndex.value % props.frames.length;
  return props.frames[index];
});

/** The current playback timecode in seconds. */
const timecode = computed(() => {
  if (props.frames.length === 0 || props.duration === 0) return 0;
  // Calculate time based on the current index and total duration
  return (currentFrameIndex.value / props.frames.length) * props.duration;
});

// ----------------------------------------------------------------------
// Animation Logic
// ----------------------------------------------------------------------

/** Advances the frame index and emits a time update. */
const nextFrame = () => {
  if (props.frames.length === 0) return;

  // Advance index and wrap around to 0
  currentFrameIndex.value = (currentFrameIndex.value + 1) % props.frames.length;

  // Emit time update
  emit("timeupdate", timecode.value);
};

/** Starts the frame animation loop. */
const startAnimation = () => {
  // If already running or no frames, stop
  if (isPlayingInternal.value && animationInterval.value !== null) return;
  if (props.frames.length === 0) return;

  isPlayingInternal.value = true;
  animationInterval.value = window.setInterval(nextFrame, frameDurationMs.value);
  emit("update:pause", false);
};

/** Stops the frame animation loop. */
const stopAnimation = () => {
  if (animationInterval.value !== null) {
    clearInterval(animationInterval.value);
    animationInterval.value = null;
  }
  isPlayingInternal.value = false;
  emit("update:pause", true);
};

/** Toggles playback on user interaction (e.g., clicking the frame). */
const togglePlay = () => {
  if (isPlayingInternal.value) {
    stopAnimation();
  } else {
    startAnimation();
  }
};

// ----------------------------------------------------------------------
// External Control (Exposed Functions)
// ----------------------------------------------------------------------

/** Jumps playback to a specific time. */
const seekToTime = (time: number) => {
  if (props.frames.length === 0 || props.duration === 0) return;

  // Calculate the corresponding frame index
  const newIndex = Math.floor((time / props.duration) * props.frames.length);

  // Ensure index is valid
  currentFrameIndex.value = Math.min(Math.max(0, newIndex), props.frames.length - 1);

  // Emit the new timecode (which might be slightly off due to frame quantization)
  emit("timeupdate", timecode.value);
};

// Expose public methods for the parent component to use (e.g., for Seek or Skip buttons)
defineExpose({
  seekToTime,
});

// ----------------------------------------------------------------------
// Watchers and Lifecycle Hooks
// ----------------------------------------------------------------------

/** Watch for external pause prop changes (from the parent component). */
watch(
  () => props.pause,
  (shouldPause) => {
    if (shouldPause) {
      stopAnimation();
    } else {
      // Only start if there are frames
      if (props.frames.length > 0) {
        startAnimation();
      }
    }
  },
  { immediate: true },
);

/** Watch for changes in FPS or frame array length and restart the interval. */
watch([frameDurationMs, () => props.frames.length], () => {
  if (isPlayingInternal.value && props.frames.length > 0) {
    stopAnimation();
    startAnimation();
  }
});

/** Stop animation when the component is unmounted. */
onBeforeUnmount(() => {
  stopAnimation();
});
</script>

<style scoped>
.image-frame-player {
  width: 100%;
  height: 100%;
  cursor: pointer;
  /* Add any other necessary styling to match the original video element's container */
}
</style>
