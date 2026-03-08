<template>
  <span class="progress position-relative my-1 border border-info" aria-valuemin="0" aria-valuemax="100">
    <span class="progress-bar bg-info progress-bar-striped" role="progressbar" :style="{ width: props.percent + '%' }" :aria-valuenow="props.percent" aria-valuemin="0" aria-valuemax="100"></span>
  </span>
</template>

<script setup lang="ts">
const props = defineProps<{ percent: number }>();
</script>

<style scoped></style>
