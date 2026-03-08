<template>
  <div class="table-responsive">
    <table class="table table-sm bg-white table-bordered border-dark table-hover">
      <thead>
        <tr>
          <th class="text-center px-2 bg-light">{{ t("videoView.segment.start") }}</th>
          <th class="text-center px-2 bg-light">{{ t("videoView.segment.end") }}</th>
          <th class="bg-light text-center px-2">{{ t("videoView.segment.duration") }}</th>
          <th class="bg-light text-center px-2" v-if="showDestroy" @click="removeAll">
            <i class="bi bi-trash3 text-danger" style="cursor: pointer"></i>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr style="cursor: pointer !important" class="align-middle" :class="{ 'bg-secondary': overview.marking.selected }" @click="emit('selected', overview.marking)" :key="String(overview.marking.timestart) + String(overview.marking.timeend)" v-for="overview in markingsOverview">
          <td class="px-2 text-center py-1">{{ overview.start }}</td>
          <td class="px-2 text-center py-1">{{ overview.end }}</td>
          <td class="px-2 text-center py-1">{{ overview.duration }}</td>
          <td class="text-center p-0" v-if="showDestroy">
            <button @click="emit('destroy', overview.marking)" class="btn btn-sm bg-transparent">
              <i class="bi bi-trash3 text-danger"></i>
            </button>
          </td>
        </tr>
        <tr class="bg-danger-subtle">
          <td colspan="2" class="px-2 fw-bold bg-transparent">Total (min)</td>
          <td class="px-2 fw-bold text-center bg-transparent">{{ markingsDuration }}</td>
          <td class="px-2 fw-bold text-center bg-transparent" v-if="showDestroy"></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed  } from "vue";
import { useI18n } from "vue-i18n";
import type { Selection } from "../components/VideoStripe.vue";

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const { t } = useI18n();

const emit = defineEmits<{
  (e: "selected", value: Selection): void;
  (e: "destroy", value: Selection): void;
}>();

const props = defineProps<{ markings: Selection[]; showDestroy: boolean }>();

// --------------------------------------------------------------------------------------
// Methods
// --------------------------------------------------------------------------------------

const secondsToTimeCode = (seconds: number) => {
  const date = new Date(0);
  date.setSeconds(seconds);

  const s = date.toISOString().substring(11, 19);

  if (s.startsWith("00:0")) {
    return s.substring(4);
  }

  if (s.startsWith("00:")) {
    return s.substring(3);
  }

  return s;
};

const markingsOverview = computed(() => {
  return props.markings
    .slice()
    .sort((a, b) => a.timestart - b.timestart)
    .map((x) => {
      return {
        marking: x,
        start: secondsToTimeCode(x.timestart),
        end: secondsToTimeCode(x.timeend),
        duration: secondsToTimeCode(x.timeend - x.timestart)};
    });
});

const removeAll = () => {
  markingsOverview.value.forEach((s) => emit("destroy", s.marking));
};

const markingsDuration = computed(() => {
  const totalSeconds = props.markings.reduce((acc, current) => acc + (current.timeend - current.timestart), 0);

  return secondsToTimeCode(totalSeconds);
});
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;

[data-bs-theme="light"] {
  .table .col-sorted {
    background-color: bootstrap.$light;
  }
}

[data-bs-theme="dark"],
.table-rounded th,
.table-rounded td {
  table,
  .table-rounded th,
  .table-rounded td {
    border: 1px solid bootstrap.$info;
  }

  .table .col-sorted {
    background-color: bootstrap.$primary;
  }

  table th {
    background-color: bootstrap.$primary !important;
  }
}
</style>

