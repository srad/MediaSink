<template>
    <DataTable :columns="processesCols" :data="processes">
      <template #cell-args="{ value }">
        <textarea disabled readonly class="form-control" v-model="value as string" />
      </template>
    </DataTable>
</template>

<script setup lang="ts">
import { type ServicesProcessInfo } from "../services/api/v2/MediaSinkClient";
import { onMounted, ref } from "vue";
import { createClient } from "../services/api/v2/ClientFactory";
import DataTable from "../components/DataTable.vue";
import type { Column } from "../types/table";

const processesCols: Column[] = [
  { label: "PID", key: "pid", width: "10%" },
  { label: "Path", key: "path", width: "20%" },
  { label: "Args", key: "args", width: "width: 70%" },
];

const processes = ref<ServicesProcessInfo[]>([]);

onMounted(async () => {
  await load();
});

const load = async () => {
    const client = createClient();
    processes.value = await client.processes.processesList();
};
</script>
