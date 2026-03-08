<template>
  <DataTable
    default-sort-key="isRecording"
    default-sort-order="desc"
    :page-size="10"
    :columns="[
      { key: 'preview', label: 'Preview', sortable: false, width: '1%', headerClass: 'text-center align-middle bg-light', rowClass: 'px-1 py-0 text-center align-middle' },
      { key: 'displayName', label: 'Name', isSearchable: true, sortable: true, width: '10%', headerClass: 'text-center align-middle bg-light', rowClass: 'align-middle' },
      //{ key: 'url', label: 'Link', sortable: false, isSearchable: true, width: '20%', headerClass: 'text-center align-middle bg-light', rowClass: 'align-middle' },
      { key: 'fav', label: 'Favourite?', sortable: true, width: '5%', isSearchable: true, type: 'boolean', headerClass: 'text-center align-middle bg-light', rowClass: 'text-center align-middle' },
      { key: 'isRecording', label: 'Recording?', sortable: true, isSearchable: true, type: 'boolean', width: '5%', headerClass: 'text-center align-middle bg-light', rowClass: 'text-center align-middle' },
      { key: 'recordingsCount', label: 'Items', sortable: true, width: '5%', headerClass: 'text-center align-middle bg-light', rowClass: 'text-center align-middle' },
      { key: 'size', label: 'Size', sortable: true, width: '5%', headerClass: 'text-center align-middle bg-light', rowClass: 'text-center align-middle' },
      { key: 'createdAt', label: 'Added', sortable: true, width: '10%', headerClass: 'text-center align-middle bg-light', rowClass: 'text-center align-middle' },
    ]"
    :data="channels">
    <template #header-size>Size ({{ totalSize }})</template>

    <template #header-recordingsCount>Count ({{ totalCount }})</template>

    <template #cell-createdAt="{ row }">
      {{ fromNow(row.createdAt as number) }}
    </template>

    <template #cell-preview="{ row }">
      <img alt="preview" :src="row.preview as string" class="rounded" loading="lazy" style="height: 50px; width: auto" />
    </template>

    <template #cell-size="{ row }">
      <span>{{ (row.size as number).toFixed(1) }} GB</span>
    </template>

    <template #cell-displayName="{ row }">
      <div class="d-flex justify-content-between fs-6">
        <RouterLink :to="`/channel/${row.channelId}/${row.channelName}`">
          <span>{{ row.displayName }}</span>
        </RouterLink>
        <a class="mx-2" :href="row.url as string" target="_blank">
          Link
          <i class="bi bi-link"></i>
        </a>
      </div>
    </template>

    <template #cell-fav="{ row }">
      <ChannelFavButton :bookmarked="row.fav as boolean" :channel-id="row.channelId as number" />
    </template>

    <template #cell-isRecording="{ row }">
      <div v-if="row.isRecording">
        <i class="bi text-danger blink bi-record-fill pulse"></i>
        Recording
      </div>
      <div v-else></div>
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import ChannelFavButton from "../../components/controls/ChannelFavButton.vue";
import type { ServicesChannelInfo } from "../../services/api/v1/MediaSinkClient";
import { computed, inject } from "vue";
import DataTable from "../DataTable.vue";
import { fromNow } from "../../utils/datetime";

const props = defineProps<{
  channels: ServicesChannelInfo[];
}>();

const channels = computed(() =>
  props.channels.map((channel: ServicesChannelInfo) => ({
    ...channel,
    preview: `${fileUrl}/${channel.preview}`,
    link: `/channel/${channel.channelId}/${channel.channelName}`,
    size: channel.recordingsSize / 1024 / 1024 / 1024,
    createdAt: new Date(channel.createdAt),
  })),
);

const fileUrl = inject<string>("fileUrl");

// --------------------------------------------------------------------------------------
// Computes
// --------------------------------------------------------------------------------------

const totalSize = computed(() => ((props.channels || []).map((x) => x.recordingsSize).reduce((a, b) => a + b, 0) / 1024 / 1024 / 1024 / 1024).toFixed(2) + "TB");
const totalCount = computed(() => (props.channels || []).map((x) => x.recordingsCount).reduce((a, b) => a + b, 0));
</script>

<style scoped></style>
