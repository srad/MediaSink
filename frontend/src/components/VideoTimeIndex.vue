<template>
  <div class="position-relative time-index-container user-select-none" :style="{ width: props.width + 'px' }">
    <div class="position-absolute time-index" :key="marker.time" v-for="marker in timeMarkers" :style="{ transform: `translateX(${marker.left}px)` }">
      <div class="position-relative">
        <div class="position-relative">
          <div class="position-absolute" style="transform: translateX(-50%)">{{ marker.time }}</div>
          <div class="position-absolute m-0 p-0 border-end border-1" style="border-color: deeppink !important; height: 200px; transform: translate(-50%, -100%)"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { secondsToMMSS } from "../utils/time";
import { throttle } from "lodash";

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{ duration: number; width: number }>();

// --------------------------------------------------------------------------------------
// refs
// --------------------------------------------------------------------------------------

type TimeMarker = { left: number; time: string };

const timeMarkers = ref<TimeMarker[]>([]);

// --------------------------------------------------------------------------------------
// computes
// --------------------------------------------------------------------------------------

const updateTimeMarkers = throttle(() => {
  // Scaled inverse proportional function which makes sure the distances between
  // time markers are roughly evenly and don't shrink with stripe image resize.
  const factor = (1 / (props.width / props.duration)) * 10;
  const segmentDensity = 10 * factor;
  const pixelsPerMinute = Math.floor(props.width / (props.duration / segmentDensity));
  const segmentCount = Math.floor(props.width / pixelsPerMinute);

  const result: TimeMarker[] = new Array(segmentCount).fill(0).map((_, i) => ({ left: i * pixelsPerMinute, time: secondsToMMSS(segmentDensity * i) }));

  result.push({ left: props.width - 30, time: secondsToMMSS(props.duration) });

  timeMarkers.value = result;
}, 100);

watch(
  [() => props.width, () => props.duration],
  () => {
    updateTimeMarkers();
  },
  { immediate: true },
);
</script>

<style scoped>
.time-index-container {
  color: white;
  background: rgba(0, 0, 0, 0.4);
  font-size: 0.8rem;
  height: 16px;
  line-height: 16px;
  z-index: 1;
}

.time-index {
  text-align: center;
  z-index: 2;
  height: 100%;
}
</style>
