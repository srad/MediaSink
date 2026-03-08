<template>
  <div v-if="props.busy || props.empty" class="load-indicator-center d-flex justify-content-center align-items-center">
    <div v-if="props.busy" class="text-center">
      <div class="spinner-border text-primary" role="status" style="width: 4rem; height: 4rem">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>
    <h3 v-if="props.empty && !props.busy" class="text-dark">
      {{ props.emptyText || "..." }}
    </h3>
  </div>
  <template v-else>
    <slot></slot>
  </template>
</template>

<script setup lang="ts">

const props = defineProps<{
  busy: boolean;
  empty?: boolean;
  emptyText?: string;
}>();
</script>

<style scoped>
.load-indicator-center {
  min-height: calc(100vh - 4rem);
}
</style>

