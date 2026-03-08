<template>
  <div v-if="props.checkbox" class="form-check form-switch">
    <input class="form-check-input" type="checkbox" role="switch" v-model="darkMode" id="switchCheckDefault" />
    <label class="form-check-label" for="switchCheckDefault">Dark Mode</label>
  </div>
  <button v-else @click="toggleDarkMode" :class="classes" class="btn">
    <i class="bi bi-toggles" />
  </button>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useSettingsStore } from "../stores/settings";

const props = defineProps<{ checkbox: boolean }>();

const settingsStore = useSettingsStore();

const darkMode = ref(settingsStore.isDarkMode);

const classes = computed(() => (darkMode.value ? "btn-light" : "btn-dark border-dark-subtle"));

// Apply theme when the component mounts
document.documentElement.setAttribute("data-bs-theme", darkMode.value ? "dark" : "light");

// Watch for changes
watch(darkMode, (newValue) => {
  document.documentElement.setAttribute("data-bs-theme", newValue ? "dark" : "light");
  settingsStore.updateTheme(newValue ? "dark" : "light");
});

// Toggle function
const toggleDarkMode = () => {
  darkMode.value = !darkMode.value;
};
</script>
