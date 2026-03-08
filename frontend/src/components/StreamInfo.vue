<template>
  <ul class="list-group list-group-flush">
    <!--
    <li v-if="false" class="list-group-item d-flex justify-content-between bg-info-light bg-gradient">
      <div>
        <span class="badge me-2 user-select-none" :class="{ 'bg-danger text-white border border-danger blink': channel.isRecording, 'bg-light text-primary border-info border': !channel.isRecording }">Recording</span>
        <span class="badge user-select-none" :class="{ 'bg-success text-white border border-success': channel.isOnline, 'bg-light text-primary border-info border': !channel.isOnline }">Online</span>
      </div>
    </li v-if="false">
    -->

    <!-- tags -->
    <li class="list-group-item px-2 py-1">
      <div class="d-flex overflow-x-auto">
        <template v-if="!showTagInput && tagArray">
          <div v-for="tag in tagArray" class="d-flex gap-1 badge align-items-center bg-secondary text-dark rounded-1 me-1 user-select-none" :key="tag">
            <span @click="router.push({ query: { tag } })">{{ tag }}</span>
            <i class="bi bi-x" @click="destroyTag(tag)"></i>
          </div>
        </template>
        <div v-if="!showTagInput" style="cursor: pointer" class="badge bg-primary border-dark border rounded-1 d-flex gap-1 align-items-center" @click="showTagInput = true">
          <span>Tag</span>
          <span>+</span>
        </div>
      </div>

      <div v-show="showTagInput" class="input-group input-group-sm">
        <form @submit.prevent="addTag">
          <div class="input-group">
            <input :disabled="processingTag" ref="tagInput" class="form-control form-control-sm border-primary" v-model.lazy="tagVal" type="text" :name="`${channel.channelId}_tag`" autocapitalize="off" autocomplete="off" />
            <button type="submit" class="btn btn-sm btn-primary" :disabled="processingTag">save</button>
          </div>
        </form>
      </div>
    </li>
    <!-- /tags -->

    <li class="list-group-item streaminfo-footer d-flex justify-content-between fs-6 rounded-bottom-2">
      <div class="d-flex w-75 gap-2">
        <span class="form-check form-switch">
          <input @click="emit('pause', channel)" class="form-check-input" type="checkbox" :checked="!channel.isPaused" :id="`${channel.channelId}_isPaused`" :name="`${channel.channelId}_isPaused`" />
          <label class="form-check-label" :for="`${channel.channelId}_isPaused`">Record</label>
        </span>
        <FavButton :data="channel" :faved="fav" @fav="emit('unfav', channel)" @unfav="emit('fav', channel)" />
      </div>

      <div class="d-flex justify-content-evenly w-25 gap-2">
        <a @click="emit('edit', channel)">
          <i class="bi bi-pencil-fill"></i>
        </a>
        <a class="text-danger" @click="emit('destroy', channel)">
          <i class="bi bi-trash-fill"></i>
        </a>
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useRouter } from "vue-router";
import type { ServicesChannelInfo as ChannelResponse } from "../services/api/v1/MediaSinkClient";
import { validTag } from "../utils/parser";
import FavButton from "./controls/FavButton.vue";
import { createClient } from "../services/api/v1/ClientFactory";

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  fav: boolean;
  channel: ChannelResponse;
}>();

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{
  (e: "unfav", value: ChannelResponse): void;
  (e: "fav", value: ChannelResponse): void;
  (e: "edit", value: ChannelResponse): void;
  (e: "destroy", value: ChannelResponse): void;
  (e: "pause", value: ChannelResponse): void;
}>();

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const tagArray = ref<string[]>(props.channel.tags || []);
const tagVal = ref("");
const showTagInput = ref(false);
const tagInput = ref<HTMLInputElement | null>(null);
const processingTag = ref(false);
const router = useRouter();

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(showTagInput, (val) => {
  if (val) {
    tagInput.value?.focus();
  }
});

// --------------------------------------------------------------------------------------
// Methods
// --------------------------------------------------------------------------------------

const destroyTag = async (tag: string) => {
  const removeTag = tagArray.value?.filter((t) => t !== tag);
  const client = createClient();
  await client.channels.tagsPartialUpdate({ id: props.channel.channelId! }, { tags: removeTag });
  tagArray.value = removeTag;
};

const addTag = async () => {
  try {
    const tag = tagVal.value.trim().toLowerCase();

    // No value, cancel
    if (tag.trim() === "") {
      showTagInput.value = false;
      return;
    }

    if (!validTag(tag)) {
      throw new Error(`Illegal tag: ${tag}`);
    }

    processingTag.value = true;

    const client = createClient();
    await client.channels.tagsPartialUpdate({ id: props.channel.channelId! }, { tags: tagArray.value.concat(tag) });
    tagArray.value.push(tag);
    showTagInput.value = false;
    tagVal.value = "";
  } catch (e: unknown) {
    alert(e);
  } finally {
    processingTag.value = false;
    tagVal.value = "";
  }
};
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;

[data-bs-theme="light"] {
  .streaminfo-footer {
    background: bootstrap.$secondary;
  }
}

[data-bs-theme="dark"] {
  .streaminfo-footer {
    background: bootstrap.$primary;
    color: bootstrap.$white;

    .form-switch .form-check-input:checked {
      background-color: bootstrap.$success;
      border: none #30d158;
    }

    .form-switch .form-check-input:not(:checked) {
    }
  }
}
</style>
