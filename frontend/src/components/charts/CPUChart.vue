<template>
  <div class="w-100 h-100" style="min-height: 300px" ref="container"></div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, useTemplateRef, watch } from "vue";
import type { IChartApi, ISeriesApi } from "lightweight-charts";
import { createChart, LineSeries } from "lightweight-charts";

const props = defineProps<{
  series: { load: number; time: number }[];
}>();

const container = useTemplateRef<HTMLDivElement>("container");
let chart: IChartApi | null = null;
let cpuSeries: ISeriesApi<"Line"> | null = null;

const getIn = computed(() =>
  (props.series || [])
    .map((x, i) => ({
      time: i,
      value: x.load,
    }))
    .slice(-30),
);

watch(
  () => props.series,
  () => {
    //@ts-expect-error just overkill
    cpuSeries?.setData(getIn.value.sort((a, b) => a.time - b.time));
    chart?.timeScale().fitContent();
  },
);

onMounted(() => {
  chart = createChart(container.value!, {
    autoSize: true,
    layout: { attributionLogo: false, fontSize: 11 },
    localization: {
      priceFormatter: (value: number) => {
        return value.toFixed(1) + "%";
      },
    },
  });
  cpuSeries = chart?.addSeries(LineSeries, {
    pointMarkersVisible: true,
    lineWidth: 2,
    title: "Load",
  });
});

onUnmounted(() => {
  if (chart) {
    chart.remove();
    chart = null;
  }
});
</script>

<style scoped></style>
