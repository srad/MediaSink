<template>
  <div class="toast-container position-fixed top-0 end-0 p-2">
    <div :id="`toast_${toast.id}`" v-for="toast in toasts" :key="toast.id" class="toast show rounded-3 border-1" :class="toastClass[toast.kind]" role="alert" aria-live="assertive" aria-atomic="true">
      <div class="toast-header rounded-top-2">
        <strong class="me-auto">{{ toast.title }}</strong>
        <button type="button" class="btn-close" @click="emit('destroy', toast)" aria-label="Close"></button>
      </div>
      <div class="toast-body rounded-bottom-2">
        <div>
          {{ toast.message }}
        </div>
        <div style="height: 3px" :style="{ width: toast.countdown + '%' }" class="mt-2 line-indicator"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { type Toast, ToastKind } from "../stores/toast";
import { computed } from "vue";

// --------------------------------------------------------------------------------------
// Emit
// --------------------------------------------------------------------------------------

const emit = defineEmits<{ (e: "destroy", value: Toast): void }>();

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{ toasts: Toast[] }>();

// --------------------------------------------------------------------------------------
// Computes
// --------------------------------------------------------------------------------------

const toasts = computed(() => props.toasts.filter((toast) => !toast.hide));

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const toastClass: { [key in ToastKind]: string } = {
  [ToastKind.Info]: "info",
  [ToastKind.Warning]: "warning",
  [ToastKind.Error]: "danger",
  [ToastKind.Success]: "success",
};
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;

@mixin toast-variant($name, $border-color, $bg-color, $text-color: bootstrap.$light) {
  .toast.#{$name} {
    border: 1px solid $border-color;

    .toast-header {
      background-color: $bg-color;
      color: $text-color;
    }

    .line-indicator {
      background-color: $bg-color;
    }
  }
}

@include toast-variant("info", bootstrap.$info, bootstrap.$info, bootstrap.$dark);
@include toast-variant("warning", bootstrap.$warning, bootstrap.$warning, bootstrap.$dark);
@include toast-variant("success", bootstrap.$success, bootstrap.$success, bootstrap.$light);
@include toast-variant("danger", bootstrap.$danger, bootstrap.$danger, bootstrap.$light);
</style>
