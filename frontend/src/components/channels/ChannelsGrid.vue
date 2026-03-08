<template>
  <div class="my-3 mx-2">
    <div class="row g-4">
      <!-- Loop through the images array and generate the image collage -->
      <div v-for="(channel, index) in props.channels" :key="index" class="col-12 col-md-6 col-lg-4 col-xxl-3">
        <div class="collage-item">
          <RouterLink :to="`/channel/${channel.channelId}/${channel.channelName}`">
            <img loading="lazy" alt="preview" :src="`${fileUrl}/${channel.preview}`" />
          </RouterLink>
          <div class="recording-indicator" v-if="channel.isRecording">
            <i class="bi fs-5 text-danger blink bi-record-fill pulse"></i>
          </div>
          <div class="favorite-btn">
            <ChannelFavButton :bookmarked="channel.fav" :channel-id="channel.channelId" @click.stop />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ServicesChannelInfo } from "../../services/api/v1/MediaSinkClient";
import ChannelFavButton from "../../components/controls/ChannelFavButton.vue";
import { inject } from "vue";

const fileUrl = inject("fileUrl") as string;

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  channels: ServicesChannelInfo[];
}>();
</script>

<style scoped>
.collage-item {
  position: relative;
  overflow: hidden;
  border-radius: 5px;
}

.collage-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.collage-item:hover {
  transform: scale(1.05);
  transition: all 0.3s ease;
}

.favorite-btn {
  position: absolute;
  bottom: 7%;
  right: 5%;
  background-color: rgba(0, 0, 0, 0.5);
  border: none;
  border-radius: 50%;
  padding: 5px 10px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.favorite-btn:hover {
  background-color: rgba(0, 0, 0, 0.7);
}

.recording-indicator {
  position: absolute;
  top: 5%;
  left: 5%;
}
</style>
