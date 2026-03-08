<template>
  <a style="color: deeppink" :disabled="props.disabled">
    <i class="bi bi-heart-fill heart" @click="emit('fav', props.data, $event)" v-if="faved" @click.stop></i>
    <i v-else class="bi bi-heart heart" @click="emit('unfav', props.data, $event)" @click.stop></i>
  </a>
</template>

<script setup lang="ts" generic="T">
const emit = defineEmits<{
  (e: "fav", data: T, event: Event): void;
  (e: "unfav", data: T, event: Event): void;
}>();

const props = defineProps<{
  faved: boolean;
  data: T;
  disabled?: boolean;
}>();
</script>

<style scoped>
.heart {
  color: deeppink;
}
</style>
