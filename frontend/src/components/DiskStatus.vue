<template>
  <div class="text-white d-flex gap-2 justify-content-between justify-content-center mx-2">
    <i class="bi bi-sd-card-fill text-info"></i>
    <span class="progress my-1 position-relative" style="min-width: 80px">
      <span class="progress-bar bg-info progress-bar-striped" role="progressbar" :style="{ width: props.pcent + '%' }" :aria-valuenow="props.pcent" aria-valuemin="0" aria-valuemax="100"></span>
    </span>
    <span style="font-size: 90%">{{ props.pcent }}%</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ pcent: number }>();
</script>

<style scoped></style>
