<template>
  <div class="timeline-container" ref="timelineRef" @mousedown="startSelection">
    <div class="timeline-scroll" ref="scrollRef">
      <div v-for="(selection, index) in selections" :key="index" class="selection" :style="selectionStyle(selection)">
        <span class="selection-duration">{{ formatDuration(selection) }}</span>
        <button class="remove-btn" @mousedown.stop @click.stop="removeSelection(index)">×</button>
        <div class="handle left" @mousedown.stop="startResize(selection, 'left')"></div>
        <div class="handle right" @mousedown.stop="startResize(selection, 'right')"></div>
      </div>
      <div v-for="(marker, index) in timeMarkers" :key="index" class="time-marker" :style="markerStyle(marker)"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";

type Selection = {
  start: number;
  end: number;
  startTime: number;
  endTime: number;
};

const props = defineProps<{ duration: number }>();
const emit = defineEmits<{ (event: "updateSelections", selections: Selection[]): void }>();

const timelineRef = ref<HTMLDivElement | null>(null);
const scrollRef = ref<HTMLDivElement | null>(null); // Reference to the scrollable container
const imageRef = ref<HTMLImageElement | null>(null);
const selections = ref<Selection[]>([]);
const isSelecting = ref<boolean>(false);
const isResizing = ref<boolean>(false);
const currentSelection = ref<Selection | null>(null);
const resizeDirection = ref<"left" | "right" | null>(null);

const imageNaturalWidth = ref(0); // Variable to store the image's actual width

// Convert pixel to time based on the actual width of the image
const convertToTime = (pixel: number) => (pixel / imageNaturalWidth.value) * props.duration;

// Convert time to pixel based on the actual width of the image
const convertToPixel = (time: number) => (time / props.duration) * imageNaturalWidth.value;

const timeMarkers = computed(() => {
  const markers = [];
  for (let i = 5; i <= props.duration; i += 5) {
    markers.push(i);
  }
  return markers;
});

// Start selection based on mouse position and scroll position
const startSelection = (event: MouseEvent) => {
  if (isResizing.value) {
    return;
  }
  isSelecting.value = true;
  const rect = timelineRef.value?.getBoundingClientRect();
  if (!rect || !timelineRef.value || !scrollRef.value || !imageRef.value) {
    return;
  }

  const scrollLeft = scrollRef.value.scrollLeft; // Get the scroll position of the scrollable container
  const startX = event.clientX - rect.left + scrollLeft; // Correctly calculate the start position by adding scrollLeft

  const newSelection: Selection = {
    start: startX,
    end: startX,
    startTime: convertToTime(startX),
    endTime: convertToTime(startX),
  };

  selections.value.push(newSelection);
  currentSelection.value = newSelection;
};

// Update the selection position based on mouse movement and scroll position
const updateSelection = (event: MouseEvent) => {
  if (!isSelecting.value || !currentSelection.value) {
    return;
  }
  const rect = timelineRef.value?.getBoundingClientRect();
  if (!rect || !timelineRef.value || !scrollRef.value || !imageRef.value) {
    return;
  }

  const scrollLeft = scrollRef.value.scrollLeft; // Get the current scroll position of the scrollable container
  const endX = event.clientX - rect.left + scrollLeft; // Correctly calculate the end position by adding scrollLeft

  // Update the selection bounds
  currentSelection.value.end = Math.max(endX, currentSelection.value.start);
  currentSelection.value.startTime = convertToTime(currentSelection.value.start);
  currentSelection.value.endTime = convertToTime(currentSelection.value.end);

  // Only emit if the selection duration is at least 1 second
  if (currentSelection.value.endTime - currentSelection.value.startTime >= 1) {
    emit("updateSelections", selections.value);
  }
};

// End selection when the mouse is released
const endSelection = () => {
  if (currentSelection.value && currentSelection.value.endTime - currentSelection.value.startTime < 1) {
    selections.value.pop(); // Remove selection if it's shorter than 1 second
  }
  isSelecting.value = false;
  currentSelection.value = null;
};

// Remove a selection
const removeSelection = (index: number) => {
  selections.value.splice(index, 1);
  emit("updateSelections", selections.value);
};

// Style for selection
const selectionStyle = (selection: Selection) => ({
  left: `${selection.start - (scrollRef.value?.scrollLeft || 0)}px`, // Apply scrollLeft adjustment here
  width: `${selection.end - selection.start}px`,
});

// Style for time markers
const markerStyle = (time: number) => ({
  left: `${convertToPixel(time)}px`,
});

// Format the selection duration as a string
const formatDuration = (selection: Selection) => {
  const duration = selection.endTime - selection.startTime;
  const minutes = Math.floor(duration / 60);
  const seconds = (duration % 60).toFixed(1);
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
};

// Add event listeners for mouse events
onMounted(() => {
  window.addEventListener("mousemove", updateSelection);
  window.addEventListener("mouseup", endSelection);
});

// Resize handling functions
const startResize = (selection: Selection, direction: "left" | "right") => {
  isResizing.value = true;
  resizeDirection.value = direction;
  currentSelection.value = selection;
};

const updateResize = (event: MouseEvent) => {
  if (!isResizing.value || !currentSelection.value || !scrollRef.value) {
    return;
  }
  const rect = timelineRef.value?.getBoundingClientRect();
  if (!rect) {
    return;
  }

  const scrollLeft = scrollRef.value.scrollLeft; // Get the scroll position of the scrollable container
  const moveX = event.clientX - rect.left + scrollLeft;

  if (resizeDirection.value === "left") {
    currentSelection.value.start = Math.min(moveX, currentSelection.value.end);
  } else if (resizeDirection.value === "right") {
    currentSelection.value.end = Math.max(moveX, currentSelection.value.start);
  }

  // Update the selection times
  currentSelection.value.startTime = convertToTime(currentSelection.value.start);
  currentSelection.value.endTime = convertToTime(currentSelection.value.end);

  // Only emit if the selection duration is at least 1 second
  if (currentSelection.value.endTime - currentSelection.value.startTime >= 1) {
    emit("updateSelections", selections.value);
  }
};

// End resize when mouse is released
const endResize = () => {
  isResizing.value = false;
  resizeDirection.value = null;
  currentSelection.value = null;
};

onMounted(() => {
  window.addEventListener("mousemove", updateResize);
  window.addEventListener("mouseup", endResize);
});
</script>

<style scoped>
.timeline-container {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.timeline-scroll {
  position: relative;
  width: 100%;
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  white-space: nowrap;
}

.selection {
  position: absolute;
  top: 0;
  height: 100%;
  background: rgba(0, 123, 255, 0.4);
  border: 1px solid #007bff;
  cursor: pointer;
}

.selection-duration {
  position: absolute;
  top: 5px;
  left: 5px;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  padding: 2px 5px;
  font-size: 12px;
  border-radius: 3px;
}

.remove-btn {
  position: absolute;
  top: 5px;
  right: 5px;
  background: red;
  color: white;
  border: none;
  padding: 2px 6px;
  cursor: pointer;
  font-size: 14px;
  border-radius: 3px;
}

.time-marker {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: rgba(255, 255, 255, 0.7);
}

.handle {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 5px;
  background: rgba(0, 0, 0, 0.5);
  cursor: pointer;
}

.left {
  left: 0;
}

.right {
  right: 0;
}
</style>
