<template>
  <div ref="stripeContainer" @mousedown.stop="startSelection" class="position-relative h-100 user-select-none overflow-x-auto" draggable="false" style="min-height: 100px">
    <div ref="imageRowContainer" draggable="false" class="d-flex flex-nowrap h-100 position-relative">
      <img v-for="(frame, i) in frames" :key="frame" draggable="false" loading="lazy" :alt="String(i)" @load="updateWidth" :src="frame" style="height: 100%; display: block" :style="{ width: imageWidth > 0 ? imageWidth + 'px' : 'auto' }" />

      <!-- Scene Boundaries Overlay - positioned inside scrollable content -->
      <div v-if="props.scenes && props.scenes.length > 0 && width > 0" class="position-absolute top-0 start-0 h-100" :style="{ width: width + 'px', pointerEvents: 'none' }">
        <div
          v-for="(scene, index) in props.scenes"
          :key="`scene-${index}`"
          class="position-absolute border-start border-2 scene-boundary"
          :style="{
            left: (scene.startTime / props.duration) * width + 'px',
            height: '100%',
            borderColor: getSceneColor(scene.changeIntensity),
            opacity: 0.8
          }"
          :title="`Scene ${index + 1}: ${scene.startTime.toFixed(1)}s - ${scene.endTime.toFixed(1)}s\nIntensity: ${scene.changeIntensity.toFixed(2)}`">
        </div>
      </div>

      <!-- Highlight Markers Overlay - positioned inside scrollable content -->
      <div v-if="props.highlights && props.highlights.length > 0 && width > 0" class="position-absolute top-0 start-0 h-100" :style="{ width: width + 'px', pointerEvents: 'none' }">
        <div
          v-for="(highlight, index) in props.highlights"
          :key="`highlight-${index}`"
          class="position-absolute highlight-marker"
          :style="{
            left: (highlight.timestamp / props.duration) * width + 'px',
            bottom: '0',
            width: '3px',
            height: getHighlightHeight(highlight.intensity),
            backgroundColor: getHighlightColor(highlight.intensity),
            opacity: 0.7
          }"
          :title="`Motion at ${highlight.timestamp.toFixed(1)}s\nIntensity: ${highlight.intensity.toFixed(2)}\nType: ${highlight.type}`">
        </div>
      </div>
    </div>

    <div class="position-absolute bottom-0" style="pointer-events: none">
      <VideoTimeIndex v-if="imageLoaded && props.loaded" :duration="props.duration" :width="width" />
    </div>

    <div
      :key="selection.start"
      @click="select(i)"
      class="marking position-absolute"
      v-for="(selection, i) in drawSelections"
      :style="{
        transform: `translateX(${selection.start}px)`,
        width: selection.end - selection.start + 'px',
        height: '100%',
        top: 0,
      }">
      <div class="selection w-100 h-100" style="pointer-events: none" :class="{ selected: selection.selected }"></div>
      <div v-if="(currentSelection !== null && hasMinSelection) || !currentSelection" class="handle handle-left position-absolute" @mousedown.stop="startResize(selection, 'left')"></div>
      <div v-if="(currentSelection !== null && hasMinSelection) || !currentSelection" class="handle handle-right position-absolute" @mousedown.stop="startResize(selection, 'right')"></div>
      <div v-if="(currentSelection === selection && hasMinSelection) || currentSelection !== selection" class="selection-duration">{{ formatDuration(selection) }}</div>
      <button type="button" v-if="currentSelection !== selection" @click.stop="destroy(i)" class="text-white btn btn-danger p-1 bi bi-x op-100 marking-destroy position-absolute"></button>
    </div>

    <div v-if="showBar" class="timecode position-absolute" :style="{ transform: `translateX(${barLeft}px)`, zIndex: 40 }"></div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from "vue";
import { animateScrollLeft } from "../utils/animations";
import VideoTimeIndex from "./VideoTimeIndex.vue";
import BigNumber from "bignumber.js";

export type Selection = {
  selected?: boolean;
  start: number;
  end: number;
  timestart: number;
  timeend: number;
};

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  loaded: boolean;
  frames: string[];
  timecode: number;
  duration: number;
  paused: boolean;
  disabled?: boolean;
  seeked: number;
  scenes?: Array<{ startTime: number; endTime: number; changeIntensity: number }>;
  highlights?: Array<{ timestamp: number; intensity: number; type: string }>;
}>();

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{
  (e: "seek", timeIndex: number): void;
  (e: "selecting"): void;
  (e: "marking", value: Selection[]): void;
}>();

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const selections = ref<Selection[]>([]);
const isSelecting = ref<boolean>(false);
const currentSelection = ref<Selection | null>(null);
const isResizing = ref<boolean>(false);
const resizeDirection = ref<"left" | "right" | null>(null);
const showBar = ref(true);

const imageRowContainer = ref<HTMLElement | null>(null);
const stripeContainer = ref<HTMLElement>();

const width = ref(0);
const barLeft = ref(0);
const imageLoaded = ref(false);
const imageWidth = ref(0); // Width of each individual image in pixels

let seekedThroughStripeClick = false;
let rafPending = false; // RequestAnimationFrame throttle flag

const minDuration = 1;

// --------------------------------------------------------------------------------------
// Computed
// --------------------------------------------------------------------------------------

const dT = computed<number>(() => {
  if (currentSelection.value) {
    return currentSelection.value.timeend - currentSelection.value.timestart;
  }
  return 0;
});

const hasMinSelection = computed<boolean>(() => dT.value > minDuration);

const drawSelections = computed(() => (currentSelection.value ? selections.value.concat(currentSelection.value) : selections.value));

// --------------------------------------------------------------------------------------
// Methods - Defined before watchers to avoid hoisting issues
// --------------------------------------------------------------------------------------

const updateWidth = () => {
  if (imageRowContainer.value) {
    width.value = imageRowContainer.value.scrollWidth;
    if (width.value > 0 && !imageLoaded.value) {
      imageLoaded.value = true;
    }
  }
};

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(
  () => props.seeked,
  (timeIndex: number) => {
    if (props.disabled || !imageLoaded.value || !stripeContainer.value) {
      return;
    }

    if (seekedThroughStripeClick) {
      seekedThroughStripeClick = false;
      return;
    }

    const timeOffset = timeIndex / props.duration;
    barLeft.value = new BigNumber(timeOffset).multipliedBy(width.value).toNumber();

    const scrollLeft = barLeft.value - stripeContainer.value.getBoundingClientRect().width / 2;
    animateScrollLeft(stripeContainer.value, scrollLeft, 1000);
  },
);

watch(
  () => props.timecode,
  (timecode) => {
    requestAnimationFrame(() => {
      if (!imageLoaded.value || width.value === 0) return;
      const timeOffset = timecode / props.duration;
      barLeft.value = new BigNumber(timeOffset).multipliedBy(width.value).toNumber();
    });
  },
);

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onMounted(() => {
  // --- ADDED: Wheel event listener for zooming ---
  stripeContainer.value?.addEventListener("wheel", resizePreview, { passive: false });

  window.addEventListener("mousemove", updateResize);
  window.addEventListener("mouseup", endResize);
  window.addEventListener("mousemove", updateSelection);
  window.addEventListener("mouseup", endSelection);
});

onUnmounted(() => {
  stripeContainer.value?.removeEventListener("wheel", resizePreview);

  window.removeEventListener("mousemove", updateResize);
  window.removeEventListener("mouseup", endResize);
  window.removeEventListener("mousemove", updateSelection);
  window.removeEventListener("mouseup", endSelection);
});

// --------------------------------------------------------------------------------------
// More Methods
// --------------------------------------------------------------------------------------

const getCurrentTimeIndex = (): BigNumber => {
  const offset = new BigNumber(barLeft.value).dividedBy(width.value);
  return offset.multipliedBy(props.duration);
};

const emitCurrentTimeIndex = (): void => {
  emit("seek", getCurrentTimeIndex().toNumber());
};

const seek = (event: MouseEvent): void => {
  if (props.disabled || !imageLoaded.value || !stripeContainer.value) {
    return;
  }
  seekedThroughStripeClick = true;
  barLeft.value = stripeContainer.value.scrollLeft + getX(event);
  showBar.value = true;
  emitCurrentTimeIndex();
};

const startResize = (selection: Selection, direction: "left" | "right") => {
  isResizing.value = true;
  resizeDirection.value = direction;
  currentSelection.value = selection;
};

const endResize = () => {
  if (isSelecting.value) {
    return;
  }
  isResizing.value = false;
  resizeDirection.value = null;
  currentSelection.value = null;
};

const updateResize = (event: MouseEvent) => {
  if (!isResizing.value || !currentSelection.value) {
    return;
  }
  const x = getMouseX(event);
  if (resizeDirection.value === "left") {
    currentSelection.value.start = Math.min(x, currentSelection.value.end);
    currentSelection.value.timestart = convertToTime(x);
  } else if (resizeDirection.value === "right") {
    currentSelection.value.end = Math.max(x, currentSelection.value.start);
    currentSelection.value.timeend = convertToTime(x);
  }
};

const formatDuration = (selection: Selection) => {
  const duration = selection.timeend - selection.timestart;
  const minutes = Math.floor(duration / 60);
  const seconds = (duration % 60).toFixed(1);
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
};

const destroy = (index: number): void => {
  if (props.disabled) {
    return;
  }
  selections.value.splice(index, 1);
  emit("marking", selections.value);
};

const getMouseX = (event: MouseEvent): number => {
  if (!stripeContainer.value) {
    return 0;
  }
  // This is the mouse position relative to the *entire scrolled content*
  return event.clientX - stripeContainer.value.getBoundingClientRect().left + stripeContainer.value.scrollLeft;
};

const getX = (event: MouseEvent): number => {
  const bounds = stripeContainer.value!.getBoundingClientRect();
  // This is the mouse position relative to the *visible viewport*
  return event.clientX - bounds.left;
};

const startSelection = (event: MouseEvent): void => {
  if (isResizing.value) {
    return;
  }
  const target = event.target as HTMLElement;
  if (target.classList.contains("handle") || target.classList.contains("selection") || target.classList.contains("marking-destroy")) {
    return;
  }
  isSelecting.value = true;
  const posX = getMouseX(event);
  const selection = {
    selected: false,
    start: posX,
    end: posX,
    timestart: (posX / width.value) * props.duration,
    timeend: (posX / width.value) * props.duration,
  };
  currentSelection.value = selection;
};

const convertToTime = (posX: number): number => (posX / width.value) * props.duration;

const updateSelection = (event: MouseEvent) => {
  if (!isSelecting.value || !currentSelection.value) {
    return;
  }
  const endX = stripeContainer.value!.scrollLeft + getX(event);
  currentSelection.value.end = Math.max(endX, currentSelection.value.start);
  currentSelection.value.end = Math.min(width.value, currentSelection.value.end); // Guard
  currentSelection.value.timestart = convertToTime(currentSelection.value.start);
  currentSelection.value.timeend = convertToTime(currentSelection.value.end);
};

const endSelection = (event: MouseEvent): void => {
  if (!isSelecting.value || !currentSelection.value) {
    return;
  }
  if (currentSelection.value && dT.value < minDuration) {
    seek(event);
  } else {
    selections.value.push(currentSelection.value);
    emit("marking", selections.value);
  }
  isSelecting.value = false;
  currentSelection.value = null;
  emit("marking", selections.value);
};

const overlaps = (selectionStart: number, selectionEnd: number) => {
  if (!isSelecting.value) return false;
  const sorted = selections.value.sort((a, b) => a.start - b.start).sort((a, b) => a.end - b.end);
  if (selectionStart < sorted[0]!.start && selectionEnd < sorted[0]!.end) return false;
  for (let i = 0; i < sorted.length; i++) {
    const start = sorted[i]!.start;
    const end = sorted[i]!.end;
    const overlaps = (start >= selectionStart && selectionEnd <= end) || (selectionStart <= start && end >= selectionStart && end <= selectionEnd) || (start && selectionStart <= end && selectionEnd >= end) || (selectionStart <= start && selectionEnd >= end);
    if (overlaps) return true;
  }
  return false;
};

const select = (index: number): void => {
  if (props.disabled || !selections.value[index]) {
    return;
  }
  selections.value.map((x) => (x.selected = false));
  selections.value[index].selected = true;
};

// ---
// ADDED: The complete, working resizePreview function
// ---
/**
 * Get color for scene boundary based on change intensity
 */
const getSceneColor = (intensity: number): string => {
  if (intensity < 0.3) return "#ffd700"; // Gold - subtle
  if (intensity < 0.6) return "#ff8c00"; // DarkOrange - moderate
  if (intensity < 0.85) return "#ff4500"; // OrangeRed - significant
  return "#ff0000"; // Red - dramatic
};

/**
 * Get color for highlight marker based on motion intensity
 */
const getHighlightColor = (intensity: number): string => {
  if (intensity < 0.2) return "#00ff00"; // Green - minimal
  if (intensity < 0.5) return "#ffff00"; // Yellow - low-moderate
  if (intensity < 0.75) return "#ffa500"; // Orange - moderate-high
  return "#ff0000"; // Red - very high
};

/**
 * Get height for highlight marker based on motion intensity
 */
const getHighlightHeight = (intensity: number): string => {
  const minHeight = 20;
  const maxHeight = 60;
  const height = minHeight + (maxHeight - minHeight) * intensity;
  return `${height}%`;
};

const resizePreview = (event: WheelEvent): void => {
  // Check if this is a horizontal scroll event (e.g., Shift+Wheel or trackpad)
  if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
    // Let the browser handle its native horizontal scrolling
    return;
  }

  // If we are here, it's a vertical scroll. Prevent default page scroll.
  event.preventDefault();
  event.stopPropagation();

  // Throttle to one update per animation frame
  if (rafPending) return;

  if (!stripeContainer.value || !imageRowContainer.value) return;

  const oldWidth = width.value;
  if (oldWidth === 0) return;

  // --- Capture initial image width on first resize ---
  if (imageWidth.value === 0) {
    const firstImage = imageRowContainer.value.querySelector("img");
    if (firstImage) {
      imageWidth.value = firstImage.clientWidth;
    } else {
      return; // No images loaded yet
    }
  }

  const oldImageWidth = imageWidth.value;

  // Get mouse position relative to *full content*
  const mouseX = getMouseX(event);
  // Get mouse position relative to *visible viewport*
  const mouseXInViewport = getX(event);

  // --- Calculate new image width ---
  const resizeBy = event.deltaY * 1; // Increased sensitivity for faster response
  let newImageWidth = oldImageWidth + resizeBy;
  const minImageWidth = 20;
  const maxImageWidth = 500;
  newImageWidth = Math.max(minImageWidth, Math.min(maxImageWidth, newImageWidth));

  if (newImageWidth === oldImageWidth) return;

  // --- Apply new image width ---
  imageWidth.value = newImageWidth;

  // Set RAF throttle flag
  rafPending = true;

  // --- Update on next animation frame ---
  requestAnimationFrame(() => {
    // Wait for Vue to update the DOM with new image widths
    nextTick(() => {
      // Get new width after images resize
      updateWidth();
      const newWidth = width.value;

      if (newWidth !== oldWidth && newWidth > 0) {
        // Calculate scaling factor
        const factor = newWidth / oldWidth;

        // Scale all pixel-based coordinates
        barLeft.value *= factor;
        selections.value.forEach((m) => {
          m.start *= factor;
          m.end *= factor;
        });

        // Set new scroll position to keep mouse centered
        const newMouseX = mouseX * factor;
        stripeContainer.value!.scrollLeft = newMouseX - mouseXInViewport;
      }

      // Release RAF throttle flag
      rafPending = false;
    });
  });
};
</script>

<style scoped>
.marking {
  height: 100%;
}

/* z-index fix */
.selection {
  position: absolute;
  top: 0;
  height: 100%;
  background: rgba(0, 123, 255, 0.35);
  cursor: pointer;
  z-index: 10;
}

.handle {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 4px;
  opacity: 0.9;
  background: #007bff;
  cursor: ew-resize;
  user-select: none;
  z-index: 20; /* Handles on top of selection */
}

.selected {
  background: blueviolet;
  opacity: 0.4;
}

.selection-duration {
  user-select: none;
  position: absolute;
  top: 5px;
  left: 10px;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  padding: 2px 5px;
  font-size: 12px;
  border-radius: 3px;
  z-index: 30; /* Text on top */
}

.marking-destroy {
  right: 8px;
  top: 4px;
  line-height: 18px;
  margin: 0;
  z-index: 30; /* Button on top */
}

.timecode {
  width: 3px;
  height: 100%;
  background: greenyellow;
  user-select: none;
  pointer-events: none;
  top: 0; /* Explicitly set top */
  /* z-index is set inline in the template (zIndex: 40) */
}

img {
  user-select: none;
  image-rendering: optimizeSpeed;
  -webkit-user-drag: none;
  user-drag: none;
  /* Performance optimizations */
  will-change: width;
  backface-visibility: hidden;
  transform: translateZ(0);
}

.handle-left {
  left: 0;
}

.handle-right {
  right: 0;
}

/* Analysis overlay styles */
.scene-boundary {
  transition: opacity 0.2s;
}

.scene-boundary:hover {
  opacity: 1 !important;
}

.highlight-marker {
  transition: opacity 0.2s;
}

.highlight-marker:hover {
  opacity: 1 !important;
}
</style>
