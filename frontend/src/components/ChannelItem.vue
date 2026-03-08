<template>
  <div class="card rounded-2 bg-light mb-3 border shadow-sm position-relative" :class="{ 'animate__animated animate__zoomOut': destroyed, 'opacity-50 border-paused': props.channel.isPaused, 'border-recording': !props.channel.isRecording }">
    <div v-if="busy" class="bg-dark opacity-50 position-absolute w-100 h-100 d-flex align-items-center justify-content-center" style="z-index: 100">
      <div class="loader"></div>
    </div>

    <RouterLink class="text-decoration-none" :to="channelLink">
      <VideoPreview
        :min-recording="props.channel.minRecording"
        :recordings-count="props.channel.recordingsCount"
        :recordings-size="props.channel.recordingsSize"
        :is-recording="props.channel.isRecording"
        :data="props.channel.channelId"
        :preview-image="previewImage"
        @selected="viewFolder(props.channel.channelId!, props.channel.channelName)" />
    </RouterLink>

    <div class="card-body">
      <div class="card-title py-2 px-3 m-0" :class="{ 'bg-primary': !props.channel.isRecording, 'bg-danger': props.channel.isRecording }">
        <h6 class="p-0 m-0 text-white">
          <a class="text-white" target="_blank" :href="props.channel.url">
            {{ props.channel.displayName }}
            <i class="bi bi-link" />
          </a>
        </h6>
      </div>
    </div>
    <StreamInfo :channel="props.channel" :fav="props.channel.fav" @edit="(data) => emit('edit', data)" @fav="fav" @unfav="unfav" @pause="pause" @destroy="destroyChannel" />
  </div>
</template>

<script setup lang="ts">
import { inject, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

import { createClient } from "../services/api/v1/ClientFactory";
import type { ServicesChannelInfo as ChannelInfo } from "../services/api/v1/MediaSinkClient";

import StreamInfo from "./StreamInfo.vue";
import VideoPreview from "./VideoPreview.vue";

import { useChannelStore } from "../stores/channel";

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{ (e: "edit", value: ChannelInfo): void }>();

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{ channel: ChannelInfo }>();

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const channelStore = useChannelStore();
const { t } = useI18n();

const router = useRouter();

const fileUrl = inject<string>("fileUrl");

const destroyed = ref(false);
const busy = ref(false);

const channelLink = `/channel/${props.channel.channelId}/${props.channel.channelName}`;
const previewImage = `${fileUrl}/${props.channel.preview}`;

// --------------------------------------------------------------------------------------
// Methods
// --------------------------------------------------------------------------------------

const fav = async (channel: ChannelInfo) => {
  const client = createClient();
  await client.channels.favPartialUpdate({ id: channel.channelId! });
  channelStore.fav(channel.channelId);
};

const unfav = async (channel: ChannelInfo) => {
  const client = createClient();
  await client.channels.unfavPartialUpdate({ id: channel.channelId! });
  channelStore.unfav(channel.channelId);
};

const destroyChannel = async (channel: ChannelInfo) => {
  if (window.confirm(t("crud.destroy", [channel.channelName]))) {
    try {
      const client = createClient();
      busy.value = true;
      await client.channels.channelsDelete({ id: channel.channelId! });
      destroyed.value = true;
      setTimeout(() => {
        channelStore.destroy(channel.channelId);
      }, 1000);
    } catch (ex) {
      alert(ex);
    } finally {
      busy.value = false;
    }
  }
};

const pause = async (channel: ChannelInfo) => {
  try {
    const client = createClient();
    busy.value = true;
    const method = channel.isPaused ? client.channels.resumeCreate : client.channels.pauseCreate;
    await method({ id: channel.channelId! });

    // Invert current paused mode.
    channelStore[channel.isPaused ? "resume" : "pause"](channel.channelId);
  } catch (err) {
    console.log(err);
  } finally {
    busy.value = false;
  }
};

const viewFolder = (id: number, name: string) => router.push(`/channel/${id}/${name}`);
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;
@use "sass:color";

[data-bs-theme="light"] {
  .card {
    border-color: bootstrap.$primary !important;
  }
}

[data-bs-theme="dark"] {
  .card {
    border-color: color.mix(white, bootstrap.$primary, 30%) !important;
  }
}
</style>
