import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/library_controller.dart";
import "package:mediasink_app/app/models.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

List<DbRecording> _videos(int count) {
  return List<DbRecording>.generate(
    count,
    (index) => sampleVideo(id: index + 1),
    growable: false,
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  test("recording add inserts at the top on latest page one with createdAt desc", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      latestVideos: <DbRecording>[sampleVideo(id: 1), sampleVideo(id: 2)],
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = LibraryController(api: api, socket: socket);
    await controller.ready;

    socket.emit(
      SocketEventMessage(
        name: "recording:add",
        data: sampleVideo(id: 99).toJson(),
      ),
    );
    await waitForAsyncTasks();

    expect(controller.videos.first.recordingId, 99);
    expect(controller.totalCount, 3);
  });

  test("recording add outside the inline-insert mode debounces into a refresh", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      latestVideos: <DbRecording>[sampleVideo(id: 1)],
      bookmarkedVideos: <DbRecording>[sampleVideo(id: 1)],
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = LibraryController(api: api, socket: socket);
    await controller.ready;
    await controller.setSection(LibrarySection.bookmarks);

    final baselineCalls = api.getBookmarkedVideosCalls;
    socket.emit(
      SocketEventMessage(
        name: "recording:add",
        data: sampleVideo(id: 3).toJson(),
      ),
    );
    await waitForDebounce();

    expect(api.getBookmarkedVideosCalls, baselineCalls + 1);
  });

  test("deleting the last item on a later page rewinds to the previous page", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      latestVideos: _videos(26),
    );
    final controller = LibraryController(api: api);
    await controller.ready;
    await controller.setPageSize(25);
    await controller.goToPage(2);

    expect(controller.currentPage, 2);
    expect(controller.videos, hasLength(1));

    await controller.deleteVideo(controller.videos.single);

    expect(api.deleteVideoCalls, 1);
    expect(controller.currentPage, 1);
    expect(controller.videos, hasLength(25));
  });

  test("bookmark channel filter clears when its last video is removed", () async {
    final bookmarked = <DbRecording>[
      sampleVideo(id: 1, channelName: "Alpha"),
      sampleVideo(id: 2, channelName: "Beta"),
    ];
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      bookmarkedVideos: bookmarked,
    );
    final controller = LibraryController(api: api);
    await controller.ready;
    await controller.setSection(LibrarySection.bookmarks);

    controller.setBookmarkChannelFilter("Beta");
    await waitForAsyncTasks();
    expect(controller.bookmarkChannelFilter, "Beta");

    final betaVideo = controller.videos.singleWhere((video) => video.channelName == "Beta");
    await controller.toggleBookmark(betaVideo);

    expect(controller.bookmarkChannelFilter, isEmpty);
    expect(controller.bookmarkChannels, <String>["Alpha"]);
  });

  test("library preferences survive controller recreation", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      latestVideos: _videos(40),
      randomVideos: _videos(10),
    );
    final first = LibraryController(api: api);
    await first.ready;

    await first.applyLatestFilters(
      sortColumn: RequestsVideoSortColumn.size,
      sortOrder: ModelsSortOrder.asc,
      pageSize: 25,
    );
    await first.goToPage(2);
    await first.setRandomLimit(50);

    final second = LibraryController(api: api);
    await second.ready;

    expect(second.sortColumn, RequestsVideoSortColumn.size);
    expect(second.sortOrder, ModelsSortOrder.asc);
    expect(second.pageSize, 25);
    expect(second.skip, 25);
    expect(second.randomLimit, 50);
  });

  test("multiple recording or job events debounce into a single refresh", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      latestVideos: _videos(3),
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = LibraryController(api: api, socket: socket);
    await controller.ready;

    final baselineCalls = api.filterVideosCalls;
    socket.emit(const SocketEventMessage(name: "job:done", data: null));
    socket.emit(const SocketEventMessage(name: "recording:delete", data: 1));
    socket.emit(const SocketEventMessage(name: "job:preview:done", data: 1));
    await waitForDebounce();

    expect(api.filterVideosCalls, baselineCalls + 1);
  });
}
