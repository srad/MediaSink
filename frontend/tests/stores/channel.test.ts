import { describe, it, expect, beforeEach } from "vitest";
import { useChannelStore } from "../../src/stores/channel";
import { createPinia, setActivePinia } from "pinia";
import type { DatabaseRecording } from "../../src/services/api/v1/MediaSinkClient";

describe("Channel Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it("should initialize with an empty array of channels", () => {
    const store = useChannelStore();
    expect(store.channels).toEqual([]);
  });

  it("should fetch and set the list of channels from the API", async () => {
    const store = useChannelStore();
    await store.load();

    expect(store.channels).toHaveLength(2);
    expect(store.channels[0].channelName).toBe("Channel 1");
    expect(store.channels[1].channelName).toBe("Channel 2");
  });

  it("should return all channels", () => {
    const store = useChannelStore();

    store.channels = [
      { channelId: 1, channelName: "B", createdAt: "2024-01-01", deleted: false, displayName: "B", fav: false, isOnline: false, isPaused: false, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
      { channelId: 2, channelName: "A", createdAt: "2024-01-02", deleted: false, displayName: "A", fav: false, isOnline: false, isPaused: false, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
    ];

    expect(store.all).toEqual([
      { channelId: 1, channelName: "B", createdAt: "2024-01-01", deleted: false, displayName: "B", fav: false, isOnline: false, isPaused: false, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
      { channelId: 2, channelName: "A", createdAt: "2024-01-02", deleted: false, displayName: "A", fav: false, isOnline: false, isPaused: false, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
    ]);
  });

  it("should return not recording streams", () => {
    const store = useChannelStore();

    store.channels = [
      { channelId: 1, channelName: "channel_a", createdAt: "2025-03-23T18:36:12.266Z", deleted: false, displayName: "Channel A", fav: false, isOnline: false, isPaused: false, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
      { channelId: 2, channelName: "channel_b", createdAt: "2023-19-23T18:36:12.266Z", deleted: false, displayName: "Channel B", fav: false, isOnline: false, isPaused: false, isRecording: true, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
    ];

    expect(store.notRecordingStreams).toEqual([{ channelId: 1, channelName: "channel_a", createdAt: "2025-03-23T18:36:12.266Z", deleted: false, displayName: "Channel A", fav: false, isOnline: false, isPaused: false, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" }]);
  });

  it("should return disabled streams", () => {
    const store = useChannelStore();

    store.channels = [
      { channelId: 1, channelName: "B", createdAt: "2024-01-01", deleted: false, displayName: "B", fav: false, isOnline: false, isPaused: true, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
      { channelId: 2, channelName: "A", createdAt: "2024-01-02", deleted: false, displayName: "A", fav: false, isOnline: false, isPaused: false, isRecording: true, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
    ];

    expect(store.disabledStreams).toEqual([{ channelId: 1, channelName: "B", createdAt: "2024-01-01", deleted: false, displayName: "B", fav: false, isOnline: false, isPaused: true, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" }]);
  });

  it("should return recording streams", () => {
    const store = useChannelStore();

    store.channels = [
      { channelId: 1, channelName: "B", createdAt: "2024-01-01", deleted: false, displayName: "B", fav: false, isOnline: false, isPaused: true, isRecording: false, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
      { channelId: 2, channelName: "A", createdAt: "2024-01-02", deleted: false, displayName: "A", fav: false, isOnline: false, isPaused: false, isRecording: true, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" },
    ];

    expect(store.recordingStreams).toEqual([{ channelId: 2, channelName: "A", createdAt: "2024-01-02", deleted: false, displayName: "A", fav: false, isOnline: false, isPaused: false, isRecording: true, isTerminating: false, minDuration: 0, minRecording: 0, preview: "", recordingsCount: 0, recordingsSize: 0, skipStart: 0, tags: [], url: "" }]);
  });

  it("should add a recording", () => {
    const store = useChannelStore();

    store.channels = [
      {
        channelId: 1,
        channelName: "B",
        createdAt: "2024-01-01",
        deleted: false,
        displayName: "B",
        fav: false,
        isOnline: false,
        isPaused: true,
        isRecording: false,
        isTerminating: false,
        minDuration: 0,
        minRecording: 0,
        preview: "",
        recordingsCount: 0,
        recordingsSize: 0,
        skipStart: 0,
        tags: [],
        url: "",
      },
    ];

    const recording: DatabaseRecording = {
      bitRate: 1000,
      bookmark: true,
      channelId: 1,
      channelName: "B",
      createdAt: "2024-01-03",
      duration: 60,
      filename: "test.mp4",
      height: 720,
      packets: 1000,
      pathRelative: "/path/to/file.mp4",
      previewCover: "",
      previewStripe: "",
      previewVideo: "",
      recordingId: 3,
      size: 1000000,
      videoType: "h264",
      width: 1280,
    };

    store.addRecording(recording);

    expect(store.channels[0].recordings).toContainEqual(recording);
  });
});
