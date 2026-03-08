<template>
  <div class="row">
    <div class="col-md-12">
      <div class="card border-primary shadow-sm mb-3">
        <div class="card-header py-2 px-3 bg-primary-subtle d-flex gap-2">
          <div>CPU Load</div>
          <div v-if="!cpuInfo" class="spinner-grow my-1 text-primary spinner-grow-sm" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
          <i v-else class="bi bi-cpu-fill"></i>
        </div>
        <div class="card-body p-1">
          <CPUChart :series="cpuLoadSeries" />
        </div>
      </div>
    </div>

    <div class="col-md-12">
      <div class="card border-primary shadow-sm">
        <div class="card-header py-2 px-3 bg-primary-subtle d-flex gap-2">
          <div>Network Traffic</div>
          <div v-if="!netInfo" class="spinner-grow my-1 text-primary spinner-grow-sm" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
          <i v-else class="bi bi-hdd-network-fill"></i>
        </div>
        <div class="card-body p-1">
          <TrafficChart :series="trafficSeries" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HelpersCPUInfo, HelpersDiskInfo, HelpersNetInfo } from "@/services/api/v1/MediaSinkClient";
import TrafficChart from "@/components/charts/TrafficChart.vue";
import CPUChart from "@/components/charts/CPUChart.vue";
import { onMounted, ref } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { createClient } from "@/services/api/v1/ClientFactory";

const importing = ref(false);
const importProgress = ref(0);
const importSize = ref(0);

const trafficSeries = ref<{ in: number; out: number; time: number }[]>([]);
const cpuLoadSeries = ref<{ load: number; time: number }[]>([]);

const cpuInfo = ref<HelpersCPUInfo | undefined>(undefined);
const diskInfo = ref<HelpersDiskInfo | undefined>(undefined);
const netInfo = ref<HelpersNetInfo | undefined>(undefined);

const id = ref<number>(0);

const fetch = async () => {
  try {
    const client = createClient();
    const data = await Promise.all([client.info.infoDetail({ seconds: 1 }), client.admin.importList()]);

    if (data) {
      netInfo.value = data[0].netInfo;
      cpuInfo.value = data[0].cpuInfo;
      diskInfo.value = data[0].diskInfo;

      trafficSeries.value = trafficSeries.value.concat({
        in: data[0].netInfo.receiveBytes / 1024 / 1024,
        out: data[0].netInfo.transmitBytes / 1024 / 1024,
        time: Date.now(),
      });
      cpuLoadSeries.value = cpuLoadSeries.value.concat({
        load: data[0].cpuInfo.loadCpu[0]!.load * 100,
        time: Date.now(),
      });

      importing.value = data![1].isImporting || false;
      importProgress.value = data![1].progress || 0;
      importSize.value = data![1].size || 0;
    }
  } catch (err) {
    alert(err);
  }
};

onBeforeRouteLeave(() => {
  clearInterval(id.value);
});

onMounted(async () => {
  await fetch();
  id.value = setInterval(fetch, 2500);
});
</script>

<style scoped></style>
