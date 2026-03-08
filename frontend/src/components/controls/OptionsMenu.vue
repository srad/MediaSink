<template>
  <div class="d-flex gap-2 align-items-center">
    <button type="button" class="btn btn-info btn-sm d-flex gap-2" @click="clickFile">
      <input ref="file" name="file" v-show="false" accept="video/mp4" @change="fileChanged" type="file" />
      <span class="d-none d-sm-inline">Upload video</span>
      <i class="bi bi-upload" />
    </button>
    <button type="button" class="btn btn-info btn-sm d-flex gap-2" @click="emit('edit')">
      <span class="d-none d-sm-inline">Edit channel</span>
      <i class="bi bi-pencil" />
    </button>
    <button type="button" class="btn btn-danger btn-sm d-flex gap-2" @click="emit('delete')">
      <span class="d-none d-sm-inline">Delete channel</span>
      <i class="bi bi-trash3-fill" />
    </button>
    <div class="form-check form-switch d-flex justify-content-between" v-if="multiSelect">
      <label class="form-check-label m-0 p-0 me-2" for="flexSwitchCheckDefault">Enabled?</label>
      <input class="form-check-input bg-info border-info m-0 my-1 p-0" type="checkbox" role="switch" id="flexSwitchCheckDefault" :checked="!props.channelPaused" @change="emit('pause', $event.target as HTMLInputElement)" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, useTemplateRef } from "vue";

const props = defineProps<{
  channelPaused: boolean;
  multiSelect: boolean;
}>();

const show = ref(false);

const emit = defineEmits<{
  (e: "file", value: File): void;
  (e: "pause", value: HTMLInputElement): void;
  (e: "edit", value: void): void;
  (e: "delete", value: void): void;
}>();

const file = useTemplateRef<HTMLInputElement>("file");

const clickFile = () => file.value?.click();

const fileChanged = () => {
  const el = file.value;

  if (el && el.files && el.files.length > 0) {
    const firstFile = el.files[0];
    if (firstFile) {
      emit("file", firstFile);
      // clear old file
      el.value = "";
    }
  }
};
</script>

<style scoped>
@use "@/assets/custom-bootstrap.scss" as bootstrap;

[data-bs-theme="light"] {
  .card {
    border-color: bootstrap.$primary !important;
  }
}

[data-bs-theme="dark"] {
  .card {
    border-color: bootstrap.$info !important;
  }

  input {
    color: bootstrap.$primary;
  }
}
</style>
