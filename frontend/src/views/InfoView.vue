<template>
  <div class="container-fluid py-2">
    <div class="table-responsive">
      <table class="table table-bordered">
        <thead>
          <tr style="visibility: collapse">
            <th style="width: 25%"></th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td colspan="2" class="p-0"></td>
          </tr>
          <tr v-for="version in versions" :key="version[0]">
            <td class="bg-light-subtle align-middle">
              {{ version[0] }}
            </td>
            <td class="align-middle">{{ version[1] }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HelpersCPUInfo, HelpersDiskInfo, HelpersNetInfo } from "@/services/api/v1/MediaSinkClient";
import { inject, onMounted, type Ref, ref } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { createClient } from "@/services/api/v1/ClientFactory";

const build = inject("build") as string;
const version = inject("version") as string;

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
    }
  } catch (err) {
    alert(err);
  }
};

onBeforeRouteLeave(() => {
  clearInterval(id.value);
});

const versions: Ref<[string, string][]> = ref([]);

onMounted(async () => {
  const client = createClient();
  const data = await client.admin.versionList();

  versions.value.push(["Client-Version", version]);
  versions.value.push(["Client-Revision", build]);
  versions.value.push(["Server-Version", data.version]);
  versions.value.push(["Server-Revision", data.commit]);

  id.value = setInterval(fetch, 2500);
});
</script>

<style scoped></style>
