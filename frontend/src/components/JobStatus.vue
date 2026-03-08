<template>
  <button type="button" class="d-flex position-relative gap-2 btn btn-info" @click="emit('click')">
    <span>{{ props.totalCount }}</span>
    <span>{{ t("menu.jobs") }}</span>
  </button>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const props = defineProps<{ totalCount: number }>();
const emit = defineEmits<{ (e: "click"): void }>();
</script>
