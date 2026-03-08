<template>
  <LoadIndicator :busy="isLoading">
    <div class="pb-2">
      <div class="mb-2 px-0">
        <div class="d-flex justify-content-between">
          <div class="d-flex">
            <button @click="() => settingsStore.setChannelsLayout(ChannelsViewLayout.Grid)" type="button" class="btn btn-sm me-2" :class="{ 'btn-success': settingsStore.isChannelsGridLayout, 'btn-secondary': settingsStore.isChannelsListLayout }">
              <i class="bi bi-grid"></i>
            </button>
            <button @click="() => settingsStore.setChannelsLayout(ChannelsViewLayout.List)" type="button" class="btn btn-sm me-2" :class="{ 'btn-secondary': settingsStore.isChannelsGridLayout, 'btn-success': settingsStore.isChannelsListLayout }">
              <i class="bi bi-list"></i>
            </button>
          </div>
          <div class="d-flex">
            <button @click="downloadChannelsAsJson" type="button" class="btn btn-sm btn-primary me-2">Export channels</button>
            <button @click="inputFileClick" type="button" class="btn btn-sm btn-primary">Import channels</button>
            <input ref="channelsFile" accept="application/json" type="file" name="importChannels" hidden @change="inputFileChanged" />
          </div>
        </div>
        <div v-if="isImporting">
          <h6>Importing ...</h6>
          <div class="progress" role="progressbar" aria-label="Animated striped example" aria-valuenow="75" aria-valuemin="0" aria-valuemax="100">
            <div class="progress-bar progress-bar-striped progress-bar-animated" style="width: 75%"></div>
          </div>
          <hr />
        </div>
      </div>

      <FillNotice v-if="channelStore.all.length == 0">
        <h1>No Channels</h1>
      </FillNotice>

      <ChannelsList v-if="settingsStore.isChannelsListLayout" :channels="channelStore.all" />
      <ChannelsGrid v-else :channels="channelStore.all" />
    </div>
  </LoadIndicator>
</template>

<script setup lang="ts">
import { onMounted, ref, useTemplateRef } from "vue";

import type { DatabaseChannel } from "@/services/api/v1/MediaSinkClient";
import { createClient } from "@/services/api/v1/ClientFactory";
import { downloadObjectAsJson } from "@/utils/file";
import LoadIndicator from "@/components/LoadIndicator.vue";
import { ChannelsViewLayout, useSettingsStore } from "@/stores/settings.ts";
import ChannelsGrid from "@/components/channels/ChannelsGrid.vue";
import ChannelsList from "@/components/channels/ChannelsList.vue";
import { useChannelStore } from "@/stores/channel.ts";
import FillNotice from "@/components/FillNotice.vue";
import { useToastStore } from "@/stores/toast.ts";

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------
const channelStore = useChannelStore();
const toastStore = useToastStore();

// --------------------------------------------------------------------------------------
// refs
// --------------------------------------------------------------------------------------

const isImporting = ref(false);
const channelsFile = useTemplateRef<HTMLInputElement>("channelsFile");
const isLoading = ref(true);

const settingsStore = useSettingsStore();

// --------------------------------------------------------------------------------------
// Functions
// --------------------------------------------------------------------------------------

const downloadChannelsAsJson = async () => {
  try {
    const client = createClient();
    const channels = await client.channels.channelsList();
    downloadObjectAsJson(channels, "channels", document.body);
  } catch (error) {
    alert(error);
  }
};

const inputFileClick = () => channelsFile.value?.click();

const inputFileChanged = (event: Event) => {
  const target = event.target as HTMLInputElement;

  if (!target.files?.length) {
    return;
  }

  const file = target.files[0];

  file
    ?.text()
    .then(JSON.parse)
    .then((channels) => importChannels(channels));
};

const importChannels = async (channelsResponse: DatabaseChannel[]) => {
  isImporting.value = true;
  const client = createClient();

  const results = await Promise.allSettled(
    channelsResponse.map((channel) =>
      client.channels.channelsCreate({
        minDuration: channel.minDuration,
        channelName: channel.channelName,
        displayName: channel.displayName,
        isPaused: channel.isPaused,
        skipStart: channel.skipStart,
        tags: channel.tags,
        url: channel.url,
      }),
    ),
  );

  let successCount = 0;
  let failureCount = 0;

  results.forEach((result, index) => {
    if (result.status === "fulfilled") {
      channelStore.add(result.value);
      successCount++;
    } else {
      failureCount++;
      console.error(`Failed to import channel "${channelsResponse[index]?.displayName || channelsResponse[index]?.channelName}":`, result.reason);
    }
  });

  isImporting.value = false;

  if (successCount > 0 && failureCount === 0) {
    toastStore.success({ title: "Import Complete", message: `Successfully imported ${successCount} channel${successCount > 1 ? "s" : ""}` });
  } else if (successCount > 0 && failureCount > 0) {
    toastStore.warn({ title: "Import Partially Complete", message: `Imported ${successCount} channel${successCount > 1 ? "s" : ""}, ${failureCount} failed` });
  } else if (failureCount > 0) {
    toastStore.error({ title: "Import Failed", message: `Failed to import ${failureCount} channel${failureCount > 1 ? "s" : ""}` });
  }
};

// --------------------------------------------------------------------------------------
// Hooks
// --------------------------------------------------------------------------------------

onMounted(async () => {
  await channelStore.load();
  isLoading.value = false;
});
</script>
