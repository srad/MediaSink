<template>
  <Transition name="modal">
    <div v-if="props.show" ref="container" class="modal-mask d-flex align-items-center p-3">
      <div class="modal-container">
        <div class="modal-header d-flex justify-content-between px-3 py-2">
          <slot name="header">default header</slot>
          <button type="button" class="btn-close btn-close-white" @click="emit('close')"></button>
        </div>

        <div class="modal-body m-0 p-3">
          <slot name="body">default body</slot>
        </div>

        <div v-if="props.showFooter" class="modal-footer mx-3 mb-3 rounded-bottom-2">
          <slot name="footer">
            <button class="modal-default-button btn btn-primary" @click="emit('close')">OK</button>
          </slot>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

import { ref, watch } from "vue";

interface Props {
  show: boolean;
  scrollTop?: boolean;
  showFooter?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showFooter: true,
});

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{ (e: "close"): void }>();

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const container = ref<HTMLDivElement | null>(null);

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(
  () => props.scrollTop,
  (val) => {
    if (val === true) {
      container.value?.scrollTo({
        top: 0,
        behavior: "smooth",
      });
    }
  },
);
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;

.modal-mask {
  position: fixed;
  z-index: 9998;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  transition: opacity 0.3s ease;
  overflow: auto;
}

.modal-container {
  width: 600px;
  margin: auto;
  height: fit-content;
  background-color: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.33);
  transition: all 0.3s ease;
}

.modal-header h3 {
  margin-top: 0;
  color: #42b983;
}

.modal-body {
  margin: 20px 0;
}

.modal-enter-from .modal-container,
.modal-leave-to .modal-container {
  -webkit-transform: scale(1.1);
  transform: scale(1.1);
}

[data-bs-theme="light"] {
  .modal-container {
    border: 2px solid bootstrap.$primary;
    background-color: bootstrap.$light;
  }

  .modal-header {
    background-color: bootstrap.$primary;
    margin: 0;
    padding: 0;
    color: bootstrap.$light;
  }
}

[data-bs-theme="dark"] {
  .modal-container {
    border: 1px solid bootstrap.$primary;
    background-color: bootstrap.$dark;
  }

  .modal-header {
    background-color: bootstrap.$primary;
    margin: 0;
    padding: 0;
    color: bootstrap.$dark;
  }
}
</style>
