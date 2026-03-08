<template>
  <RouterLink class="navbar-brand text-white d-flex gap-2 p-0 m-0 align-items-center" to="/streams/live">
    <img class="icon" alt="icon" src="/icons2/icon.png" />
    <span class="d-none d-md-inline">{{ props.title }}</span>
  </RouterLink>
</template>

<script setup lang="ts">

const props = defineProps<{ title: string }>();
</script>

<style>
.icon {
  display: inline-block;
  height: 40px;
  width: auto;
  animation: rotate 15s linear infinite; /* Animation for rotation */
}

@keyframes rotate {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(-360deg);
  }
}
</style>

