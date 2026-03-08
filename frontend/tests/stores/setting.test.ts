import { beforeEach, describe, expect, it } from "vitest";
import { ChannelsViewLayout, useSettingsStore } from "../../src/stores/settings";
import { createPinia, setActivePinia } from "pinia"; // Adjust the import path according to your project structure

describe("useSettingsStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it("should initialize the state correctly", () => {
    const store = useSettingsStore();
    expect(store.videoVolume).toBe(1.0);
    expect(store.videoMuted).toBe(true);
    expect(store.layout).toBe(ChannelsViewLayout.Grid);
    expect(store.filterViewPageSize).toBe(100);
    expect(store.theme).toBe("light");
  });

  it("should mute the video", () => {
    const store = useSettingsStore();
    store.mute();
    expect(store.videoMuted).toBe(true);
  });

  it("should unmute the video", () => {
    const store = useSettingsStore();
    store.unmute();
    expect(store.videoMuted).toBe(false);
  });

  it("should update volume correctly", () => {
    const store = useSettingsStore();
    store.updateVolume(0.5);
    expect(store.videoVolume).toBe(0.5);

    store.updateVolume(-1);
    expect(store.videoVolume).toBe(0);

    store.updateVolume(2);
    expect(store.videoVolume).toBe(1);
  });

  it("should set channels layout", () => {
    const store = useSettingsStore();
    store.setChannelsLayout(ChannelsViewLayout.List);
    expect(store.layout).toBe(ChannelsViewLayout.List);

    store.setChannelsLayout(ChannelsViewLayout.Grid);
    expect(store.layout).toBe(ChannelsViewLayout.Grid);
  });

  it("should update the theme", () => {
    const store = useSettingsStore();
    store.updateTheme("light");
    expect(store.theme).toBe("light");

    store.updateTheme("dark");
    expect(store.theme).toBe("dark");
  })

  it("should set filter view page size", () => {
    const store = useSettingsStore();
    store.setFilterViewPageSize(50);
    expect(store.filterViewPageSize).toBe(50);

    store.setFilterViewPageSize(200);
    expect(store.filterViewPageSize).toBe(200);
  });

  it("should return the correct isMuted state", () => {
    const store = useSettingsStore();
    store.videoMuted = true;
    expect(store.isMuted).toBe(true);

    store.videoMuted = false;
    expect(store.isMuted).toBe(false);
  });

  it("should return the current volume", () => {
    const store = useSettingsStore();
    store.videoVolume = 0.7;
    expect(store.currentVolume).toBe(0.7);

    store.videoVolume = 0.3;
    expect(store.currentVolume).toBe(0.3);
  });

  it("should return the correct isChannelsGridLayout state", () => {
    const store = useSettingsStore();
    store.layout = ChannelsViewLayout.Grid;
    expect(store.isChannelsGridLayout).toBe(true);

    store.layout = ChannelsViewLayout.List;
    expect(store.isChannelsGridLayout).toBe(false);
  });

  it("should return the correct isChannelsListLayout state", () => {
    const store = useSettingsStore();
    store.layout = ChannelsViewLayout.List;
    expect(store.isChannelsListLayout).toBe(true);

    store.layout = ChannelsViewLayout.Grid;
    expect(store.isChannelsListLayout).toBe(false);
  });

  it("should return the filter page size", () => {
    const store = useSettingsStore();
    store.filterViewPageSize = 150;
    expect(store.filterPageSize).toBe(150);

    store.filterViewPageSize = 300;
    expect(store.filterPageSize).toBe(300);
  });
});
