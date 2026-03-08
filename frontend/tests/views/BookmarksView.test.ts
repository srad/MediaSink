import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import BookmarksView from "../../src/views/BookmarksView.vue";
import VideoItem from "../../src/components/VideoItem.vue";
import { createTestingPinia } from "@pinia/testing";
import { createI18n } from "vue-i18n";
import { ComponentPublicInstance } from "vue";
import { DatabaseRecording } from "../../src/services/api/v1/MediaSinkClient";

// Define the component instance type
type BookmarksViewInstance = ComponentPublicInstance<{
  filterChannel: string;
  videos: DatabaseRecording[];
  filteredVideos: DatabaseRecording[];
  bookmark: (recording: DatabaseRecording) => void;
  destroyRecording: (recording: DatabaseRecording) => Promise<void>;
}>;

// Mock API client
const mockBookmarksList = vi.fn().mockResolvedValue([
  {
    bitRate: 1,
    bookmark: true,
    channelId: 1,
    channelName: "Channel 1",
    createdAt: Date.now().toFixed(),
    duration: 1,
    filename: "video1.mp4",
    height: 1,
    packets: 0,
    pathRelative: "",
    recordingId: 1,
    size: 0,
    videoType: "video",
    width: 0,
  },
  {
    bitRate: 1,
    bookmark: true,
    channelId: 1,
    channelName: "Channel 2",
    createdAt: Date.now().toFixed(),
    duration: 1,
    filename: "video2.mp4",
    height: 1,
    packets: 0,
    pathRelative: "",
    recordingId: 2,
    size: 0,
    videoType: "video",
    width: 0,
  },
]);

const mockRecordingsDelete = vi.fn().mockResolvedValue(true);

vi.mock("@/services/api/v1/ClientFactory", () => ({
  createClient: () => ({
    videos: {
      bookmarksList: mockBookmarksList,
      videosDelete: mockRecordingsDelete,
    },
  }),
}));

const i18n = createI18n({
  locale: "en",
  messages: { en: { "crud.destroy": "Are you sure you want to delete {0}?" } },
});

describe("BookmarksView.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders correctly with no videos", async () => {
    mockBookmarksList.mockResolvedValueOnce([]); // Simulate empty state

    const wrapper = mount(BookmarksView, {
      global: {
        plugins: [createTestingPinia(), i18n],
      },
    });

    await flushPromises(); // Wait for API call

    expect(wrapper.findAllComponents(VideoItem)).toHaveLength(0);
  });

  it("renders video items when videos are present", async () => {
    const wrapper = mount(BookmarksView, {
      global: {
        plugins: [createTestingPinia(), i18n],
      },
    });

    await flushPromises(); // Ensure videos are loaded

    expect(wrapper.findAllComponents(VideoItem)).toHaveLength(2);
  });

  it("filters videos by channel", async () => {
    const wrapper = mount(BookmarksView, {
      global: {
        plugins: [createTestingPinia(), i18n],
      },
    });

    await flushPromises();
    expect(wrapper.findAllComponents(VideoItem)).toHaveLength(2);

    const vm = wrapper.vm as unknown as BookmarksViewInstance;

    // ✅ Correct way to modify a ref() in Vue 3
    vm.filterChannel = "Channel 1"; // Access the ref proxy, Vue handles `.value` internally
    await flushPromises(); // Ensure Vue re-renders

    expect(wrapper.findAllComponents(VideoItem)).toHaveLength(1);
  });

  it("removes a video when unbookmarked", async () => {
    const wrapper = mount(BookmarksView, {
      global: {
        plugins: [createTestingPinia(), i18n],
      },
    });

    await flushPromises();
    expect(wrapper.findAllComponents(VideoItem)).toHaveLength(2);

    const vm = wrapper.vm as unknown as BookmarksViewInstance;

    vm.bookmark({
      bitRate: 1,
      bookmark: false,
      channelId: 1,
      channelName: "Channel 1",
      createdAt: Date.now().toFixed(),
      duration: 1,
      filename: "video1.mp4",
      height: 1,
      packets: 0,
      pathRelative: "",
      recordingId: 1,
      size: 0,
      videoType: "video",
      width: 0,
    });

    expect(vm.videos).toHaveLength(1);
  });

  it("confirms before deleting a recording", async () => {
    global.confirm = vi.fn(() => true);

    const wrapper = mount(BookmarksView, {
      global: {
        plugins: [createTestingPinia(), i18n],
      },
    });

    await flushPromises();
    const vm = wrapper.vm as unknown as BookmarksViewInstance;

    await vm.destroyRecording({
      bitRate: 1,
      bookmark: false,
      channelId: 1,
      channelName: "Channel 1",
      createdAt: Date.now().toFixed(),
      duration: 1,
      filename: "video1.mp4",
      height: 1,
      packets: 0,
      pathRelative: "",
      recordingId: 1,
      size: 0,
      videoType: "video",
      width: 0,
    });

    expect(global.confirm).toHaveBeenCalledWith("Are you sure you want to delete video1.mp4?");
    expect(mockRecordingsDelete).toHaveBeenCalledWith({ id: 1 });
  });
});
