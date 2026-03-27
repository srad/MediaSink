import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/jobs_controller.dart";
import "package:mediasink_app/app/models.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

List<DbJob> _jobs(int count, {DbJobStatus status = DbJobStatus.open}) {
  return List<DbJob>.generate(
    count,
    (index) => sampleJob(id: index + 1, status: status),
    growable: true,
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  test("row limit survives controller recreation", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      jobs: _jobs(10),
      workerProcessing: true,
    );
    final first = JobsController(api: api);
    await first.ready;

    await first.setRowLimit(50);

    final second = JobsController(api: api);
    await second.ready;

    expect(second.rowLimit, 50);
  });

  test("page clamps back when backend total shrinks", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      jobs: _jobs(26, status: DbJobStatus.completed),
      workerProcessing: true,
    );
    final controller = JobsController(api: api);
    await controller.ready;
    await controller.setTab(JobsTab.completed);
    await controller.goToPage(2);

    expect(controller.currentPage, 2);

    api.jobs.removeRange(25, api.jobs.length);
    await controller.refresh();

    expect(controller.currentPage, 1);
    expect(controller.totalPages, 1);
  });

  test("worker toggle pauses and resumes through the api", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      jobs: _jobs(1),
      workerProcessing: true,
    );
    final controller = JobsController(api: api);
    await controller.ready;

    expect(controller.workerProcessing, true);

    await controller.toggleWorker();
    expect(api.pauseWorkerCalls, 1);
    expect(controller.workerProcessing, false);

    await controller.toggleWorker();
    expect(api.resumeWorkerCalls, 1);
    expect(controller.workerProcessing, true);
  });

  test("deleting the last row on a later page rewinds to the previous page", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      jobs: _jobs(26, status: DbJobStatus.completed),
      workerProcessing: true,
    );
    final controller = JobsController(api: api);
    await controller.ready;
    await controller.setTab(JobsTab.completed);
    await controller.goToPage(2);

    expect(controller.currentPage, 2);
    expect(controller.jobs, hasLength(1));

    await controller.deleteJob(controller.jobs.single);

    expect(api.deleteJobCalls, 1);
    expect(controller.currentPage, 1);
    expect(controller.jobs, hasLength(25));
  });

  test("active summary counts reflect running versus queued jobs", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      jobs: <DbJob>[
        sampleJob(id: 1, status: DbJobStatus.open),
        sampleJob(id: 2, status: DbJobStatus.open),
        DbJob(
          jobId: 3,
          status: DbJobStatus.open,
          task: DbJobTask.previewFrames,
          active: false,
          channelName: "Sample Channel",
          filename: "queued.mp4",
        ),
      ],
      workerProcessing: true,
    );
    final controller = JobsController(api: api);
    await controller.ready;

    expect(controller.processingCount, 2);
    expect(controller.queuedCount, 1);
  });

  test("multiple job socket events debounce into a single refresh", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      jobs: _jobs(5),
      workerProcessing: true,
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = JobsController(api: api, socket: socket);
    await controller.ready;

    final baselineJobCalls = api.getJobsCalls;
    final baselineWorkerCalls = api.getWorkerStatusCalls;

    socket.emit(const SocketEventMessage(name: "job:created", data: null));
    socket.emit(const SocketEventMessage(name: "job:updated", data: null));
    socket.emit(const SocketEventMessage(name: "job:done", data: null));
    await waitForDebounce();

    expect(api.getJobsCalls, baselineJobCalls + 1);
    expect(api.getWorkerStatusCalls, baselineWorkerCalls + 1);
  });
}
