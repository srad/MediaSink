import { defineStore } from "pinia";
import { computed, ref } from "vue";

export enum ChannelsViewLayout {
  Grid = "grid",
  List = "list",
}

export type SettingsState = {
  videoVolume: number;
  videoMuted: boolean;
  layout: ChannelsViewLayout;
  filterViewPageSize: number;
  theme: "dark" | "light";
};

// -------------------------------------------------------------------
// Settings setup store declaration.
// All this acrobatics is only necessary because Pinia does not
// have its typesystem under control. Using the options store
// declaration leads to TS compiler errors.
// -------------------------------------------------------------------

export const useSettingsStore = defineStore(
  "settings",
  () => {
    // Individual state variables stored in refs with default values
    const videoVolume = ref(1.0);
    const videoMuted = ref(true);
    const layout = ref(ChannelsViewLayout.Grid);
    const filterViewPageSize = ref(100);
    const theme = ref<"dark" | "light">("light");

    // Actions
    const mute = () => {
      videoMuted.value = true;
    };

    const unmute = () => {
      videoMuted.value = false;
    };

    const updateVolume = (volume: number) => {
      if (volume < 0) {
        volume = 0;
      }
      if (volume > 1) {
        volume = 1;
      }
      videoVolume.value = volume;
    };

    const setChannelsLayout = (newLayout: ChannelsViewLayout) => {
      layout.value = newLayout;
    };

    const setFilterViewPageSize = (size: number) => {
      filterViewPageSize.value = size;
    };

    const updateTheme = (newTheme: "light" | "dark") => {
      theme.value = newTheme;
    };

    // Getters to access state variables
    const isMuted = computed(() => videoMuted.value);
    const isDarkMode = computed(() => theme.value === "dark");
    const isLightMode = computed(() => theme.value === "light");
    const currentVolume = computed(() => videoVolume.value);
    const isChannelsGridLayout = computed(() => layout.value === ChannelsViewLayout.Grid);
    const isChannelsListLayout = computed(() => layout.value === ChannelsViewLayout.List);
    const filterPageSize = computed(() => filterViewPageSize.value);

    return {
      videoVolume,
      videoMuted,
      layout,
      filterViewPageSize,
      theme,
      mute,
      unmute,
      updateVolume,
      setChannelsLayout,
      setFilterViewPageSize,
      updateTheme,
      isMuted,
      isDarkMode,
      isLightMode,
      currentVolume,
      isChannelsGridLayout,
      isChannelsListLayout,
      filterPageSize,
    };
  },
  {
    persist: true,
  } as any,
);
