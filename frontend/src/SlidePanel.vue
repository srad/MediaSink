<template>
  <div class="rounded-bottom-3" :class="[panelClass, 'slide-panel', positionClass]" :style="{ height: isOpen ? contentHeight : '35px', width, opacity }">
    <div ref="contentRef" class="panel-content" :style="{ maxHeight: isOpen ? contentRefHeight + 'px' : '0', opacity: isOpen ? 1 : 0, transition: 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out' }">
      <slot></slot>
    </div>
    <button class="toggle-button rounded-bottom-3" :class="[buttonClass]" @click="togglePanel">
      {{ label }}
      <i :class="isOpen ? 'bi bi-chevron-up' : 'bi bi-chevron-down'"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from "vue";

type Position = "left" | "center" | "right";

interface Props {
  label?: string;
  position?: Position;
  panelClass?: string;
  buttonClass?: string;
  width?: string;
  opacity?: number;
}

const label = computed(() => props.label || (isOpen.value ? "Close" : "Open"));
const panelClass = computed(() => props.panelClass || "bg-dark border-primary shadow-sm p-0");
const buttonClass = computed(() => props.buttonClass || "btn btn-info text-dark border-primary border-start border-end border-bottom");
const width = computed(() => props.width || "300px");
const opacity = computed(() => props.opacity || "1");

const props = defineProps<Props>();
const isOpen = ref<boolean>(false);
const contentRef = ref<HTMLElement | null>(null);
const contentHeight = ref<string>("35px");
const contentRefHeight = ref<number>(0);

const togglePanel = async (): Promise<void> => {
  isOpen.value = !isOpen.value;
  await nextTick();
  if (contentRef.value) {
    contentRefHeight.value = contentRef.value.scrollHeight;
    contentHeight.value = isOpen.value ? `${contentRefHeight.value + 32}px` : "35px";
  }
};

const positionClass = computed<string>(() => {
  return `position-${props.position || "center"}`;
});
</script>

<style scoped>
.slide-panel {
  position: fixed;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  border-top: none;
  transition: height 0.3s ease-in-out;
  z-index: 1050;
  overflow: hidden;
}

.position-left {
  left: 20px;
  transform: none;
}

.position-right {
  right: 20px;
  left: auto;
  transform: none;
}

.position-center {
  left: 50%;
  transform: translateX(-50%);
}

.toggle-button {
  width: 100%;
  color: white;
  border: none;
  padding: 0px;
  line-height: 35px;
  cursor: pointer;
  position: absolute;
  left: 0;
  bottom: 0;
  transition: bottom 0.3s ease-in-out;
}

.panel-content {
  overflow: hidden;
}
</style>
