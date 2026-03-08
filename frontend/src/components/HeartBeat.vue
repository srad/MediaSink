<template>
  <div class="mx-3 text-danger fw-bold d-none d-sm-none d-md-block">
    <i v-if="isConnected" class="bi blink bi-heart-pulse-fill" />
    <i v-else class="bi bi-heart-pulse"></i>
  </div>
</template>

<script setup lang="ts">
import { useSocketStatus } from "../composables/useSocketStatus";

const { isConnected } = useSocketStatus();
</script>
