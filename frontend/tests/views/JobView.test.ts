import { flushPromises, mount } from "@vue/test-utils";
import { createMemoryHistory, createRouter } from "vue-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestingPinia } from "@pinia/testing";
import JobView from "../../src/views/JobView.vue";
import { DbJobOrder, DbJobStatus } from "../../src/services/api/v2/MediaSinkClient";

const mockListCreate = vi.fn();
const mockWorkerList = vi.fn().mockResolvedValue({ isProcessing: true });
const mockPauseCreate = vi.fn().mockResolvedValue(undefined);
const mockResumeCreate = vi.fn().mockResolvedValue(undefined);
const mockJobsDelete = vi.fn().mockResolvedValue(undefined);
const mockSocketOn = vi.fn();
const mockSocketOff = vi.fn();

vi.mock("../../src/services/api/v2/ClientFactory", () => ({
  createClient: () => ({
    jobs: {
      listCreate: mockListCreate,
      workerList: mockWorkerList,
      pauseCreate: mockPauseCreate,
      resumeCreate: mockResumeCreate,
      jobsDelete: mockJobsDelete,
    },
  }),
}));

vi.mock("../../src/composables/useSocket", () => ({
  useSocket: () => ({
    on: mockSocketOn,
    off: mockSocketOff,
  }),
}));

const completedJob = {
  active: false,
  args: "",
  channelId: 1,
  channelName: "completed_channel",
  command: "ffmpeg ...",
  completedAt: "2026-03-19T19:54:52.173881605+01:00",
  createdAt: "2026-03-19T19:47:48.006800194+01:00",
  durationMs: 1706,
  filename: "completed.mp4",
  filepath: "/recordings/completed_channel/completed.mp4",
  info: undefined,
  jobId: 309,
  pid: undefined,
  priority: 3,
  progress: "100.00",
  recordingId: 28,
  startedAt: "2026-03-19T19:54:50.467663739+01:00",
  status: DbJobStatus.StatusJobCompleted,
  task: "analyze-frames",
};

const errorJob = {
  active: false,
  args: "",
  channelId: 5,
  channelName: "error_channel",
  command: "ffmpeg -i /recordings/error_channel/error.mp4 ...",
  completedAt: undefined,
  createdAt: "2026-03-19T16:19:08.979339204+01:00",
  durationMs: undefined,
  filename: "error.mp4",
  filepath: "/recordings/error_channel/error.mp4",
  info: "error generating preview frames: exit status 251",
  jobId: 206,
  pid: 74,
  priority: 1,
  progress: "47.36",
  recordingId: 11,
  startedAt: "2026-03-19T16:44:52.088773737+01:00",
  status: DbJobStatus.StatusJobError,
  task: "preview-frames",
};

const mountJobView = async (tab: "completed" | "error") => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/jobs/:tab", component: JobView }],
  });

  await router.push(`/jobs/${tab}`);
  await router.isReady();

  const wrapper = mount(JobView, {
    global: {
      plugins: [router, createTestingPinia({ createSpy: vi.fn })],
      stubs: {
        ModalConfirmDialog: {
          template: "<div><slot name='header' /><slot name='body' /></div>",
          props: ["show"],
        },
      },
    },
  });

  await flushPromises();
  return wrapper;
};

describe("JobView.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.removeItem("jobs-row-limit");

    mockListCreate.mockImplementation(async ({ states, sortOrder }) => {
      expect(sortOrder).toBe(DbJobOrder.JobOrderDESC);

      if (states?.[0] === DbJobStatus.StatusJobCompleted) {
        return { jobs: [completedJob], totalCount: 1, skip: 0, take: 1000 };
      }
      if (states?.[0] === DbJobStatus.StatusJobError) {
        return { jobs: [errorJob], totalCount: 1, skip: 0, take: 1000 };
      }

      return { jobs: [], totalCount: 0, skip: 0, take: 1000 };
    });
  });

  it("renders completed jobs on the completed tab", async () => {
    const wrapper = await mountJobView("completed");

    expect(mockListCreate).toHaveBeenCalledWith({
      skip: 0,
      take: 1000,
      states: [DbJobStatus.StatusJobCompleted],
      sortOrder: DbJobOrder.JobOrderDESC,
    });
    expect(wrapper.text()).toContain("completed.mp4");
    expect(wrapper.text()).toContain("Done");
    expect(wrapper.text()).not.toContain("No jobs.");
  });

  it("renders failed jobs on the error tab and shows the full error details", async () => {
    const wrapper = await mountJobView("error");

    expect(mockListCreate).toHaveBeenCalledWith({
      skip: 0,
      take: 1000,
      states: [DbJobStatus.StatusJobError],
      sortOrder: DbJobOrder.JobOrderDESC,
    });
    expect(wrapper.text()).toContain("error_channel");
    expect(wrapper.text()).toContain("Error");
    expect(wrapper.text()).toContain("error generating preview frames: exit status 251");

    await wrapper.find("button[aria-label='Inspect job 206']").trigger("click");

    expect(wrapper.text()).toContain("error.mp4");
    expect(wrapper.text()).toContain("error generating preview frames: exit status 251");
    expect(wrapper.text()).toContain("/recordings/error_channel/error.mp4");
  });
});
