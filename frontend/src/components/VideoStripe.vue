<template>
  <div ref="stripeContainer" data-testid="video-stripe-scroll" @mousedown.stop="startSelection" @scroll="handleScroll" class="position-relative h-100 user-select-none overflow-x-auto" draggable="false" style="min-height: 100px">
    <div data-testid="video-stripe-timeline" class="position-relative h-100" draggable="false" :style="{ width: `${width}px` }">
      <img
        v-for="tile in visibleTiles"
        :key="tile.key"
        draggable="false"
        loading="eager"
        decoding="async"
        :alt="String(tile.index)"
        @load="handleTileLoad"
        :src="tile.src"
        class="position-absolute top-0 timeline-tile"
        :style="{ height: '100%', left: `${tile.left}px`, width: `${tile.width + 1}px` }" />

      <div v-if="props.scenes && props.scenes.length > 0 && width > 0" class="position-absolute top-0 start-0 h-100" :style="{ width: `${width}px`, pointerEvents: 'none' }">
        <div
          v-for="(scene, index) in props.scenes"
          :key="`scene-${index}`"
          class="position-absolute border-start border-2 scene-boundary"
          :style="{
            left: `${timeToPosition(scene.startTime)}px`,
            height: '100%',
            borderColor: getSceneColor(scene.changeIntensity),
            opacity: 0.8,
          }"
          :title="`Scene ${index + 1}: ${scene.startTime.toFixed(1)}s - ${scene.endTime.toFixed(1)}s\nIntensity: ${scene.changeIntensity.toFixed(2)}`"></div>
      </div>

      <div v-if="props.highlights && props.highlights.length > 0 && width > 0" class="position-absolute top-0 start-0 h-100" :style="{ width: `${width}px`, pointerEvents: 'none' }">
        <div
          v-for="(highlight, index) in props.highlights"
          :key="`highlight-${index}`"
          class="position-absolute highlight-marker"
          :style="{
            left: `${getHighlightLeft(highlight)}px`,
            bottom: '0',
            width: `${getHighlightWidth(highlight)}px`,
            height: getHighlightHeight(highlight.intensity),
            backgroundColor: getHighlightColor(highlight.intensity),
            opacity: 0.7,
            borderRadius: highlight.startTime !== undefined && highlight.endTime !== undefined ? '3px 3px 0 0' : '0',
          }"
          :title="getHighlightTitle(highlight)"></div>
      </div>
    </div>

    <div class="position-absolute bottom-0" style="pointer-events: none">
      <VideoTimeIndex v-if="imageLoaded && props.loaded" :duration="props.duration" :width="width" />
    </div>

    <div
      v-for="(selection, index) in drawSelections"
      :key="`${selection.timestart}-${selection.timeend}-${index}`"
      @click="select(index)"
      class="marking position-absolute"
      :style="{
        transform: `translateX(${selection.start}px)`,
        width: `${selection.end - selection.start}px`,
        height: '100%',
        top: 0,
      }">
      <div class="selection w-100 h-100" style="pointer-events: none" :class="{ selected: selection.selected }"></div>
      <div v-if="(currentSelection !== null && hasMinSelection) || !currentSelection" class="handle handle-left position-absolute" @mousedown.stop="startResize(index, 'left')"></div>
      <div v-if="(currentSelection !== null && hasMinSelection) || !currentSelection" class="handle handle-right position-absolute" @mousedown.stop="startResize(index, 'right')"></div>
      <div v-if="(currentSelection === selection && hasMinSelection) || currentSelection !== selection" class="selection-duration">{{ formatDuration(selection) }}</div>
      <button type="button" v-if="currentSelection !== selection" @click.stop="destroy(index)" class="text-white btn btn-danger p-1 bi bi-x op-100 marking-destroy position-absolute"></button>
    </div>

    <div v-if="showBar" class="timecode position-absolute" :style="{ transform: `translateX(${barLeft}px)`, zIndex: 40 }"></div>
  </div>
</template>

<script setup lang="ts">
import { animateScrollLeft } from "../utils/animations";
import VideoTimeIndex from "./VideoTimeIndex.vue";
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import {
  buildVisiblePreviewTiles,
  getMaxPreviewPixelsPerSecond,
  getMinPreviewPixelsPerSecond,
  type VideoPreviewManifest,
  type VisiblePreviewTile,
} from "../utils/video";

export type Selection = {
  selected?: boolean;
  start: number;
  end: number;
  timestart: number;
  timeend: number;
};

const props = defineProps<{
  duration: number;
  disabled?: boolean;
  highlights?: Array<{ endTime?: number; intensity: number; startTime?: number; timestamp: number; type: string }>;
  loaded: boolean;
  markings?: Selection[];
  paused: boolean;
  preview?: VideoPreviewManifest | null;
  scenes?: Array<{ changeIntensity: number; endTime: number; startTime: number }>;
  seeked: number;
  timecode: number;
}>();

const emit = defineEmits<{
  (e: "marking", value: Selection[]): void;
  (e: "seek", timeIndex: number): void;
  (e: "selecting"): void;
}>();

const minDuration = 1;
const stripeContainer = ref<HTMLElement | null>(null);
const selections = ref<Selection[]>([]);
const currentSelection = ref<Selection | null>(null);
const isSelecting = ref(false);
const isResizing = ref(false);
const resizeDirection = ref<"left" | "right" | null>(null);
const currentSelectionIndex = ref<number | null>(null);
const showBar = ref(true);
const viewportWidth = ref(1);
const scrollLeft = ref(0);
const pixelsPerSecond = ref(1);
const barLeft = ref(0);
const imageLoaded = ref(false);
const zoomInitialized = ref(false);

let seekedThroughStripeClick = false;
let resizeObserver: ResizeObserver | null = null;
let rafPending = false;

const previewSource = computed<VideoPreviewManifest | null>(() => {
  if (!props.preview) {
    return null;
  }

  return {
    ...props.preview,
    duration: Math.max(props.duration || 0, props.preview.duration || 0, 1),
  };
});

const minPixelsPerSecond = computed(() => {
  if (!previewSource.value) {
    return 1;
  }

  return getMinPreviewPixelsPerSecond(previewSource.value.duration, viewportWidth.value);
});

const maxPixelsPerSecond = computed(() => {
  if (!previewSource.value) {
    return 1;
  }

  return Math.max(minPixelsPerSecond.value, getMaxPreviewPixelsPerSecond(previewSource.value.timestamps));
});

const timeline = computed(() => {
  if (!previewSource.value) {
    return {
      timelineWidth: Math.max(viewportWidth.value, 1),
      tiles: [] as VisiblePreviewTile[],
    };
  }

  return buildVisiblePreviewTiles({
    manifest: previewSource.value,
    pixelsPerSecond: pixelsPerSecond.value,
    scrollLeft: scrollLeft.value,
    viewportWidth: viewportWidth.value,
  });
});

const width = computed(() => timeline.value.timelineWidth);
const visibleTiles = computed(() => timeline.value.tiles);

const dT = computed(() => {
  if (!currentSelection.value) {
    return 0;
  }

  return currentSelection.value.timeend - currentSelection.value.timestart;
});

const hasMinSelection = computed(() => dT.value > minDuration);

const drawSelections = computed(() => (currentSelection.value ? selections.value.concat(currentSelection.value) : selections.value));

const clamp = (value: number, min: number, max: number): number => Math.min(max, Math.max(min, value));

const getContentScrollLimit = (): number => Math.max(width.value - viewportWidth.value, 0);

const timeToPosition = (timeIndex: number): number => clamp(timeIndex * pixelsPerSecond.value, 0, width.value);

const positionToTime = (position: number): number => {
  const duration = props.duration > 0 ? props.duration : previewSource.value?.duration || 0;
  if (duration <= 0 || pixelsPerSecond.value <= 0) {
    return 0;
  }

  return clamp(position / pixelsPerSecond.value, 0, duration);
};

const syncSelectionBounds = (selection: Selection): Selection => {
  const start = timeToPosition(selection.timestart);
  const end = timeToPosition(selection.timeend);
  return {
    ...selection,
    end: Math.max(start, end),
    start: Math.min(start, end),
  };
};

const syncSelectionCollection = () => {
  selections.value = selections.value.map((selection) => syncSelectionBounds(selection));
  if (currentSelection.value) {
    currentSelection.value = syncSelectionBounds(currentSelection.value);
  }
};

const emitMarkings = () => {
  emit(
    "marking",
    selections.value.map((selection) => syncSelectionBounds({ ...selection })),
  );
};

const getCurrentTimeIndex = (): number => positionToTime(barLeft.value);

const emitCurrentTimeIndex = (): void => {
  emit("seek", getCurrentTimeIndex());
};

const handleTileLoad = () => {
  if (!imageLoaded.value) {
    imageLoaded.value = true;
  }
};

const syncViewport = () => {
  if (!stripeContainer.value) {
    return;
  }

  viewportWidth.value = Math.max(stripeContainer.value.clientWidth, 1);
  scrollLeft.value = stripeContainer.value.scrollLeft;
};

const ensureZoomBounds = (resetZoom = false) => {
  if (!previewSource.value) {
    pixelsPerSecond.value = 1;
    zoomInitialized.value = false;
    return;
  }

  const nextPixelsPerSecond = resetZoom || !zoomInitialized.value ? minPixelsPerSecond.value : clamp(pixelsPerSecond.value, minPixelsPerSecond.value, maxPixelsPerSecond.value);

  pixelsPerSecond.value = nextPixelsPerSecond;
  zoomInitialized.value = true;
};

const updateBarPosition = (timeIndex: number) => {
  barLeft.value = timeToPosition(timeIndex);
};

const seek = (event: MouseEvent): void => {
  if (props.disabled || !stripeContainer.value) {
    return;
  }

  seekedThroughStripeClick = true;
  barLeft.value = stripeContainer.value.scrollLeft + getViewportX(event);
  showBar.value = true;
  emitCurrentTimeIndex();
};

const startResize = (selectionIndex: number, direction: "left" | "right") => {
  const selection = selections.value[selectionIndex];
  if (!selection) {
    return;
  }

  isResizing.value = true;
  resizeDirection.value = direction;
  currentSelectionIndex.value = selectionIndex;
  currentSelection.value = { ...selection };
};

const endResize = () => {
  if (isSelecting.value) {
    return;
  }

  if (currentSelection.value) {
    currentSelection.value = syncSelectionBounds(currentSelection.value);
    if (currentSelectionIndex.value !== null && selections.value[currentSelectionIndex.value]) {
      selections.value.splice(currentSelectionIndex.value, 1, currentSelection.value);
      emitMarkings();
    }
  }

  isResizing.value = false;
  resizeDirection.value = null;
  currentSelectionIndex.value = null;
  currentSelection.value = null;
};

const updateResize = (event: MouseEvent) => {
  if (!isResizing.value || !currentSelection.value) {
    return;
  }

  const x = getContentX(event);
  if (resizeDirection.value === "left") {
    currentSelection.value.start = Math.min(x, currentSelection.value.end);
    currentSelection.value.timestart = positionToTime(currentSelection.value.start);
  } else if (resizeDirection.value === "right") {
    currentSelection.value.end = Math.max(x, currentSelection.value.start);
    currentSelection.value.timeend = positionToTime(currentSelection.value.end);
  }
};

const formatDuration = (selection: Selection) => {
  const duration = selection.timeend - selection.timestart;
  const minutes = Math.floor(duration / 60);
  const seconds = (duration % 60).toFixed(1);
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
};

const destroy = (index: number): void => {
  if (props.disabled || !selections.value[index]) {
    return;
  }

  selections.value.splice(index, 1);
  emitMarkings();
};

const getContentX = (event: MouseEvent | WheelEvent): number => {
  if (!stripeContainer.value) {
    return 0;
  }

  return event.clientX - stripeContainer.value.getBoundingClientRect().left + stripeContainer.value.scrollLeft;
};

const getViewportX = (event: MouseEvent | WheelEvent): number => {
  if (!stripeContainer.value) {
    return 0;
  }

  return event.clientX - stripeContainer.value.getBoundingClientRect().left;
};

const startSelection = (event: MouseEvent): void => {
  if (isResizing.value || !previewSource.value) {
    return;
  }

  const target = event.target as HTMLElement;
  if (target.classList.contains("handle") || target.classList.contains("selection") || target.classList.contains("marking-destroy")) {
    return;
  }

  isSelecting.value = true;
  emit("selecting");

  const position = clamp(getContentX(event), 0, width.value);
  currentSelection.value = {
    end: position,
    selected: false,
    start: position,
    timeend: positionToTime(position),
    timestart: positionToTime(position),
  };
};

const updateSelection = (event: MouseEvent) => {
  if (!isSelecting.value || !currentSelection.value) {
    return;
  }

  const end = clamp(scrollLeft.value + getViewportX(event), 0, width.value);
  currentSelection.value.end = Math.max(end, currentSelection.value.start);
  currentSelection.value.timestart = positionToTime(currentSelection.value.start);
  currentSelection.value.timeend = positionToTime(currentSelection.value.end);
};

const endSelection = (event: MouseEvent): void => {
  if (!isSelecting.value || !currentSelection.value) {
    return;
  }

  if (dT.value < minDuration) {
    seek(event);
  } else {
    selections.value.push(syncSelectionBounds(currentSelection.value));
    emitMarkings();
  }

  isSelecting.value = false;
  currentSelection.value = null;
};

const select = (index: number): void => {
  if (props.disabled || !selections.value[index]) {
    return;
  }

  selections.value = selections.value.map((selection, selectionIndex) => ({
    ...selection,
    selected: selectionIndex === index,
  }));
};

const getSceneColor = (intensity: number): string => {
  if (intensity < 0.3) return "#ffd700";
  if (intensity < 0.6) return "#ff8c00";
  if (intensity < 0.85) return "#ff4500";
  return "#ff0000";
};

const getHighlightColor = (intensity: number): string => {
  if (intensity < 0.2) return "#00ff00";
  if (intensity < 0.5) return "#ffff00";
  if (intensity < 0.75) return "#ffa500";
  return "#ff0000";
};

const getHighlightHeight = (intensity: number): string => {
  const minHeight = 20;
  const maxHeight = 60;
  return `${minHeight + (maxHeight - minHeight) * intensity}%`;
};

const getHighlightLeft = (highlight: { endTime?: number; intensity: number; startTime?: number; timestamp: number; type: string }): number => {
  return timeToPosition(highlight.startTime ?? highlight.timestamp);
};

const getHighlightWidth = (highlight: { endTime?: number; intensity: number; startTime?: number; timestamp: number; type: string }): number => {
  if (highlight.startTime === undefined || highlight.endTime === undefined) {
    return 3;
  }
  return Math.max(timeToPosition(highlight.endTime) - timeToPosition(highlight.startTime), 4);
};

const getHighlightTitle = (highlight: { endTime?: number; intensity: number; startTime?: number; timestamp: number; type: string }): string => {
  if (highlight.startTime !== undefined && highlight.endTime !== undefined) {
    return `Motion ${highlight.startTime.toFixed(1)}s - ${highlight.endTime.toFixed(1)}s\nPeak at ${highlight.timestamp.toFixed(1)}s\nIntensity: ${highlight.intensity.toFixed(2)}\nType: ${highlight.type}`;
  }

  return `Motion at ${highlight.timestamp.toFixed(1)}s\nIntensity: ${highlight.intensity.toFixed(2)}\nType: ${highlight.type}`;
};

const handleScroll = () => {
  if (!stripeContainer.value) {
    return;
  }

  scrollLeft.value = stripeContainer.value.scrollLeft;
};

const resizePreview = (event: WheelEvent): void => {
  if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
    return;
  }

  if (!stripeContainer.value || !previewSource.value) {
    return;
  }

  event.preventDefault();
  event.stopPropagation();

  if (rafPending) {
    return;
  }

  const oldPixelsPerSecond = pixelsPerSecond.value;
  const nextPixelsPerSecond = clamp(oldPixelsPerSecond * Math.exp(-event.deltaY * 0.0025), minPixelsPerSecond.value, maxPixelsPerSecond.value);

  if (nextPixelsPerSecond === oldPixelsPerSecond) {
    return;
  }

  const anchorViewportX = getViewportX(event);
  const anchorTime = (stripeContainer.value.scrollLeft + anchorViewportX) / oldPixelsPerSecond;

  pixelsPerSecond.value = nextPixelsPerSecond;
  rafPending = true;

  requestAnimationFrame(() => {
    nextTick(() => {
      syncSelectionCollection();
      updateBarPosition(props.timecode);

      if (stripeContainer.value) {
        const nextScrollLeft = clamp(anchorTime * pixelsPerSecond.value - anchorViewportX, 0, getContentScrollLimit());
        stripeContainer.value.scrollLeft = nextScrollLeft;
        scrollLeft.value = nextScrollLeft;
      }

      rafPending = false;
    });
  });
};

watch(
  () => props.markings,
  (markings) => {
    if (isSelecting.value || isResizing.value) {
      return;
    }

    selections.value = (markings || []).map((selection) => syncSelectionBounds({ ...selection }));
  },
  { deep: true, immediate: true },
);

watch(
  [previewSource, viewportWidth],
  (newValue, oldValue) => {
    const [source, nextViewportWidth] = newValue;
    const [previousSource, previousViewportWidth] = oldValue || [];
    imageLoaded.value = !source || source.timestamps.length > 0;
    const resetZoom = source?.previewPath !== previousSource?.previewPath || nextViewportWidth !== previousViewportWidth;
    ensureZoomBounds(resetZoom && !!source);

    nextTick(() => {
      syncSelectionCollection();
      updateBarPosition(props.timecode);

      if (stripeContainer.value) {
        stripeContainer.value.scrollLeft = clamp(stripeContainer.value.scrollLeft, 0, getContentScrollLimit());
        scrollLeft.value = stripeContainer.value.scrollLeft;
      }
    });
  },
  { immediate: true },
);

watch(
  () => props.seeked,
  (timeIndex: number) => {
    if (props.disabled || !stripeContainer.value) {
      return;
    }

    if (seekedThroughStripeClick) {
      seekedThroughStripeClick = false;
      return;
    }

    updateBarPosition(timeIndex);
    const nextScrollLeft = clamp(barLeft.value - viewportWidth.value / 2, 0, getContentScrollLimit());
    animateScrollLeft(stripeContainer.value, nextScrollLeft, 300);
  },
);

watch(
  () => props.timecode,
  (timecode) => {
    requestAnimationFrame(() => {
      if (width.value === 0) {
        return;
      }

      updateBarPosition(timecode);
    });
  },
);

onMounted(() => {
  stripeContainer.value?.addEventListener("wheel", resizePreview, { passive: false });

  window.addEventListener("mousemove", updateResize);
  window.addEventListener("mouseup", endResize);
  window.addEventListener("mousemove", updateSelection);
  window.addEventListener("mouseup", endSelection);

  syncViewport();
  resizeObserver = new ResizeObserver(() => {
    syncViewport();
  });
  if (stripeContainer.value) {
    resizeObserver.observe(stripeContainer.value);
  }
});

onUnmounted(() => {
  stripeContainer.value?.removeEventListener("wheel", resizePreview);
  resizeObserver?.disconnect();
  resizeObserver = null;

  window.removeEventListener("mousemove", updateResize);
  window.removeEventListener("mouseup", endResize);
  window.removeEventListener("mousemove", updateSelection);
  window.removeEventListener("mouseup", endSelection);
});
</script>

<style scoped>
.marking {
  height: 100%;
}

.timeline-tile {
  display: block;
  object-fit: cover;
  background: rgba(0, 0, 0, 0.2);
}

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
  z-index: 20;
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
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.75rem;
  z-index: 15;
  white-space: nowrap;
}

.marking-destroy {
  top: 5px;
  right: 5px;
  z-index: 25;
  width: 24px;
  height: 24px;
  line-height: 1;
}

.handle-left {
  left: 0;
}

.handle-right {
  right: 0;
}

.timecode {
  top: 0;
  width: 2px;
  height: 100%;
  background: deepskyblue;
  pointer-events: none;
}

.highlight-marker {
  pointer-events: none;
}

.scene-boundary {
  pointer-events: none;
}
</style>
