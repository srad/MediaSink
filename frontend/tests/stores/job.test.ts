import { useJobStore } from "../../src/stores/job";
import { DatabaseJobStatus } from "../../src/services/api/v1/MediaSinkClient";
import { describe, expect, it, beforeEach } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import jobsData from "../mocks/jobs";

const mockJobs = [...jobsData.jobs];

describe("Job Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    const store = useJobStore();
    store.$reset(); // Reset the store before each test
  });

  it("should initialize with an empty array and zero jobs", () => {
    const store = useJobStore();
    expect(store.jobs).toEqual([]);
    expect(store.jobsCount).toEqual(0);
  });

  it("should return all jobs", () => {
    const store = useJobStore();
    store.jobs = [...mockJobs];
    expect(store.all).toEqual(mockJobs);
  });

  it("should load jobs from an API call", async () => {
    const store = useJobStore();
    await store.load();
    expect(store.jobs).toEqual(mockJobs);
  });

  it("should create a job", () => {
    const store = useJobStore();

    const newJob = {
      active: false,
      args: undefined,
      channelId: 3,
      channelName: "newChannel",
      command: undefined,
      completedAt: "2024-01-03T00:00:00Z",
      createdAt: "2023-12-31T00:00:00Z",
      filename: "new.mp4",
      filepath: "/path/to/new/file.mp4",
      info: undefined,
      jobId: 3,
      pid: undefined,
      progress: "25%",
      recordingId: 300,
      startedAt: "2024-01-03T00:00:00Z",
      status: DatabaseJobStatus.StatusJobOpen,
      task: {
        description: "new task",
        name: "newTask",
      },
    };

    store.add(newJob);

    expect(store.jobs).toContainEqual(newJob);
    expect(store.jobsCount).toBe(1);
  });

  it("should decrement the job count", () => {
    const store = useJobStore();
    store.jobsCount = 5;
    store.dec();
    expect(store.jobsCount).toBe(4);
  });

  it("should delete a channel's jobs", () => {
    const store = useJobStore();
    store.jobs = [...mockJobs];
    const channelIdToDelete = mockJobs[0].channelId;
    store.destroyChannel(channelIdToDelete);
    expect(store.jobs).not.toContain(mockJobs[0]);
  });

  it("should delete jobs associated with a recording", () => {
    const store = useJobStore();
    store.jobs = [...mockJobs];
    const recordingIdToDelete = mockJobs[0].recordingId;
    store.destroy(recordingIdToDelete);
    expect(store.jobs).not.toContain(mockJobs[0]);
  });

  it("should call done and decrement count after destroying a job", () => {
    const store = useJobStore();
    store.jobs = [...mockJobs];
    store.jobsCount = 3;
    const message = {
      job: mockJobs[0],
    };

    store.done(message);

    expect(store.jobs).not.toContain(mockJobs[0]);
    expect(store.jobsCount).toBe(2);
  });

  it("should set a job to inactive", () => {
    const store = useJobStore();

    const activeJob = { ...mockJobs[0], active: true };
    store.jobs = [activeJob];
    store.inactive(activeJob); // Create a copy to avoid modifying the original
    expect(store.jobs[0].active).toBe(false);
  });

  it("should update job progress", () => {
    const store = useJobStore();
    store.jobs = [...mockJobs];
    const message = {
      data: { step: 10, steps: 20, current: 50, total: 100, message: "Progressing..." },
      job: mockJobs[0],
    };
    const progress = String((((message.data.step / message.data.steps) * message.data.current) / message.data.total) * 100);

    store.progress(message);

    expect(store.jobs[0].progress).toBe(progress);
    expect(store.jobs[0].info).toBe("Progressing...");
  });

  it("should refresh jobs and total count", () => {
    const store = useJobStore();
    const newData = {
      jobs: [{ jobId: 4, channelId: 4 }],
      totalCount: 1,
    };
    store.refresh(newData);
    expect(store.jobs).toEqual([newData.jobs[0]]);
    expect(store.jobsCount).toBe(newData.totalCount);
  });

  it("should start a job and set active status", () => {
    const store = useJobStore();
    store.jobs = [...mockJobs];
    const message = {
      data: { pid: 123, command: "start" },
      job: mockJobs[0],
    };

    store.start(message);

    expect(store.jobs[0].active).toBe(true);
    expect(store.jobs[0].pid).toBe(123);
    expect(store.jobs[0].command).toBe("start");
  });
});
