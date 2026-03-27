import "dart:async";

import "package:flutter/foundation.dart";
import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/media_sink_api.dart";
import "package:mediasink_app/app/models.dart";
import "package:mediasink_app/app/session_controller.dart";
import "package:mediasink_app/app/session_storage.dart";
import "package:mediasink_app/app/socket_service.dart";

const String testServerOriginKey = "ms_server_origin";
const String testUsernameKey = "ms_username";
const String testTokenKey = "ms_token";

const AppBuildInfo testBuildInfo = AppBuildInfo(
  apiVersion: AppSessionController.supportedApiVersion,
  version: "1.0.0-test",
  build: "test-build",
);

Map<String, String> storedLoggedOutSession({
  String origin = "http://test.mediasink.local:3000",
  String username = "tester",
}) {
  return <String, String>{
    testServerOriginKey: origin,
    testUsernameKey: username,
  };
}

Map<String, String> storedAuthenticatedSession({
  String origin = "http://test.mediasink.local:3000",
  String username = "tester",
  String token = "test-token",
}) {
  return <String, String>{
    testServerOriginKey: origin,
    testUsernameKey: username,
    testTokenKey: token,
  };
}

ServerConfig testServerConfig([String origin = "http://test.mediasink.local:3000"]) {
  return ServerConfig.create(origin: origin, buildInfo: testBuildInfo);
}

ServicesChannelInfo sampleChannel({
  int id = 1,
  String name = "Sample Channel",
  bool isOnline = true,
}) {
  return ServicesChannelInfo(
    channelId: id,
    channelName: name,
    displayName: name,
    fav: true,
    isOnline: isOnline,
    isPaused: false,
    isRecording: isOnline,
    preview: "Sample Channel/.previews/live.jpg",
    tags: const <String>["test"],
    url: "https://example.com/$id",
    recordingsCount: 1,
  );
}

DbRecording sampleVideo({
  int id = 1,
  String channelName = "Sample Channel",
}) {
  return DbRecording(
    channelName: channelName,
    filename: "video_$id.mp4",
    pathRelative: "$channelName/video_$id.mp4",
    videoType: "mp4",
    bookmark: true,
    channelId: 1,
    createdAt: "2026-03-27T12:00:00Z",
    duration: 120,
    recordingId: id,
    size: 1024 * id,
  );
}

DbJob sampleJob({
  int id = 1,
  DbJobStatus status = DbJobStatus.open,
}) {
  return DbJob(
    jobId: id,
    status: status,
    task: DbJobTask.previewFrames,
    active: status == DbJobStatus.open,
    channelName: "Sample Channel",
    filename: "video_$id.mp4",
    progress: status == DbJobStatus.open ? "25" : "100",
    createdAt: "2026-03-27T12:00:00Z",
  );
}

class MemorySessionStorage implements AppSessionStorage {
  MemorySessionStorage([Map<String, String>? initialValues]) : _values = <String, String>{...?initialValues};

  final Map<String, String> _values;

  @override
  Future<String?> read({required String key}) async => _values[key];

  @override
  Future<void> write({required String key, required String value}) async {
    _values[key] = value;
  }

  @override
  Future<void> delete({required String key}) async {
    _values.remove(key);
  }
}

class FakeMediaSinkApi extends MediaSinkApi {
  FakeMediaSinkApi({
    required super.config,
    super.token,
    super.onUnauthorized,
    List<ServicesChannelInfo>? channels,
    this.recorderRunning = false,
    List<DbRecording>? latestVideos,
    List<DbRecording>? bookmarkedVideos,
    List<DbRecording>? randomVideos,
    List<DbJob>? jobs,
    this.workerProcessing = false,
    this.serverInfo = const ResponsesServerInfoResponse(version: "1.0.0-test", commit: "test-build"),
    this.loginToken = "test-token",
    this.loginError,
    this.getChannelsError,
    this.getRecorderStatusError,
    this.filterVideosError,
    this.getBookmarkedVideosError,
    this.getRandomVideosError,
    this.setVideoBookmarkError,
    this.deleteVideoError,
    this.refreshPreviewError,
    this.getJobsError,
    this.getWorkerStatusError,
    this.pauseWorkerError,
    this.resumeWorkerError,
    this.deleteJobError,
    this.pauseRecorderError,
    this.resumeRecorderError,
  }) : channels = List<ServicesChannelInfo>.of(channels ?? const <ServicesChannelInfo>[]),
       latestVideos = List<DbRecording>.of(latestVideos ?? const <DbRecording>[]),
       bookmarkedVideos = List<DbRecording>.of(bookmarkedVideos ?? const <DbRecording>[]),
       randomVideos = List<DbRecording>.of(randomVideos ?? const <DbRecording>[]),
       jobs = List<DbJob>.of(jobs ?? const <DbJob>[]);

  final List<ServicesChannelInfo> channels;
  bool recorderRunning;
  final List<DbRecording> latestVideos;
  final List<DbRecording> bookmarkedVideos;
  final List<DbRecording> randomVideos;
  final List<DbJob> jobs;
  bool workerProcessing;
  final ResponsesServerInfoResponse serverInfo;
  final String loginToken;
  final Object? loginError;
  final Object? getChannelsError;
  final Object? getRecorderStatusError;
  final Object? filterVideosError;
  final Object? getBookmarkedVideosError;
  final Object? getRandomVideosError;
  final Object? setVideoBookmarkError;
  final Object? deleteVideoError;
  final Object? refreshPreviewError;
  final Object? getJobsError;
  final Object? getWorkerStatusError;
  final Object? pauseWorkerError;
  final Object? resumeWorkerError;
  final Object? deleteJobError;
  final Object? pauseRecorderError;
  final Object? resumeRecorderError;

  int loginCalls = 0;
  int getChannelsCalls = 0;
  int getRecorderStatusCalls = 0;
  int filterVideosCalls = 0;
  int getBookmarkedVideosCalls = 0;
  int getRandomVideosCalls = 0;
  int setVideoBookmarkCalls = 0;
  int deleteVideoCalls = 0;
  int refreshPreviewCalls = 0;
  int getJobsCalls = 0;
  int getWorkerStatusCalls = 0;
  int pauseWorkerCalls = 0;
  int resumeWorkerCalls = 0;
  int deleteJobCalls = 0;
  int pauseRecorderCalls = 0;
  int resumeRecorderCalls = 0;

  void _throwIfConfigured(Object? error) {
    if (error != null) {
      throw error;
    }
  }

  @override
  Future<String> login({required String username, required String password}) async {
    loginCalls += 1;
    _throwIfConfigured(loginError);
    return loginToken;
  }

  @override
  Future<List<ServicesChannelInfo>> getChannels() async {
    getChannelsCalls += 1;
    _throwIfConfigured(getChannelsError);
    return List<ServicesChannelInfo>.of(channels);
  }

  @override
  Future<ServicesChannelInfo> getChannel(int channelId) async {
    return channels.firstWhere((channel) => channel.channelId == channelId);
  }

  @override
  Future<bool> getRecorderStatus() async {
    getRecorderStatusCalls += 1;
    _throwIfConfigured(getRecorderStatusError);
    return recorderRunning;
  }

  @override
  Future<void> pauseRecorder() async {
    pauseRecorderCalls += 1;
    _throwIfConfigured(pauseRecorderError);
    recorderRunning = false;
  }

  @override
  Future<void> resumeRecorder() async {
    resumeRecorderCalls += 1;
    _throwIfConfigured(resumeRecorderError);
    recorderRunning = true;
  }

  @override
  Future<ResponsesVideoFilterResponse> filterVideos({
    required RequestsVideoSortColumn sortColumn,
    required ModelsSortOrder sortOrder,
    int take = 100,
    int skip = 0,
  }) async {
    filterVideosCalls += 1;
    _throwIfConfigured(filterVideosError);
    final page = latestVideos.skip(skip).take(take).toList(growable: false);
    return ResponsesVideoFilterResponse(
      skip: skip,
      take: take,
      totalCount: latestVideos.length,
      videos: page,
    );
  }

  @override
  Future<List<DbRecording>> getBookmarkedVideos() async {
    getBookmarkedVideosCalls += 1;
    _throwIfConfigured(getBookmarkedVideosError);
    return List<DbRecording>.of(bookmarkedVideos);
  }

  @override
  Future<List<DbRecording>> getRandomVideos({int limit = 24}) async {
    getRandomVideosCalls += 1;
    _throwIfConfigured(getRandomVideosError);
    return randomVideos.take(limit).toList(growable: false);
  }

  @override
  Future<void> setVideoBookmark(int recordingId, bool isBookmarked) async {
    setVideoBookmarkCalls += 1;
    _throwIfConfigured(setVideoBookmarkError);
    final nextBookmark = !isBookmarked;
    _replaceVideoBookmark(latestVideos, recordingId, nextBookmark);
    final matchingVideo = latestVideos.cast<DbRecording?>().firstWhere(
      (video) => video?.recordingId == recordingId,
      orElse: () => bookmarkedVideos.cast<DbRecording?>().firstWhere(
        (video) => video?.recordingId == recordingId,
        orElse: () => null,
      ),
    );
    if (matchingVideo == null) {
      return;
    }
    final nextVideo = _copyVideoWithBookmark(matchingVideo, nextBookmark);
    bookmarkedVideos.removeWhere((video) => video.recordingId == recordingId);
    if (nextBookmark) {
      bookmarkedVideos.insert(0, nextVideo);
    }
  }

  @override
  Future<void> deleteVideo(int recordingId) async {
    deleteVideoCalls += 1;
    _throwIfConfigured(deleteVideoError);
    latestVideos.removeWhere((video) => video.recordingId == recordingId);
    bookmarkedVideos.removeWhere((video) => video.recordingId == recordingId);
    randomVideos.removeWhere((video) => video.recordingId == recordingId);
  }

  @override
  Future<void> refreshPreview(int recordingId) async {
    refreshPreviewCalls += 1;
    _throwIfConfigured(refreshPreviewError);
  }

  @override
  Future<ResponsesJobsResponse> getJobs({
    required List<DbJobStatus> states,
    required DbJobOrder sortOrder,
    int take = 100,
    int skip = 0,
  }) async {
    getJobsCalls += 1;
    _throwIfConfigured(getJobsError);
    final filtered = states.isEmpty ? jobs : jobs.where((job) => states.contains(job.status)).toList(growable: false);
    final page = filtered.skip(skip).take(take).toList(growable: false);
    return ResponsesJobsResponse(
      jobs: page,
      skip: skip,
      take: take,
      totalCount: filtered.length,
    );
  }

  @override
  Future<bool> getWorkerStatus() async {
    getWorkerStatusCalls += 1;
    _throwIfConfigured(getWorkerStatusError);
    return workerProcessing;
  }

  @override
  Future<void> pauseWorker() async {
    pauseWorkerCalls += 1;
    _throwIfConfigured(pauseWorkerError);
    workerProcessing = false;
  }

  @override
  Future<void> resumeWorker() async {
    resumeWorkerCalls += 1;
    _throwIfConfigured(resumeWorkerError);
    workerProcessing = true;
  }

  @override
  Future<void> deleteJob(int jobId) async {
    deleteJobCalls += 1;
    _throwIfConfigured(deleteJobError);
    jobs.removeWhere((job) => job.jobId == jobId);
  }

  @override
  Future<ResponsesServerInfoResponse> getServerVersion() async => serverInfo;

  void _replaceVideoBookmark(List<DbRecording> videos, int recordingId, bool bookmark) {
    for (var index = 0; index < videos.length; index += 1) {
      if (videos[index].recordingId == recordingId) {
        videos[index] = _copyVideoWithBookmark(videos[index], bookmark);
      }
    }
  }

  DbRecording _copyVideoWithBookmark(DbRecording video, bool bookmark) {
    final json = video.toJson();
    json["bookmark"] = bookmark;
    return DbRecording.fromJson(json);
  }
}

class FakeMediaSinkSocketService extends MediaSinkSocketService {
  FakeMediaSinkSocketService({
    required super.config,
    required super.token,
    SocketConnectionState initialState = SocketConnectionState.connected,
  }) : _state = initialState;

  final StreamController<SocketEventMessage> _controller = StreamController<SocketEventMessage>.broadcast();
  SocketConnectionState _state;

  @override
  Stream<SocketEventMessage> get events => _controller.stream;

  @override
  SocketConnectionState get state => _state;

  void emit(SocketEventMessage message) {
    _controller.add(message);
  }

  void setConnectionState(SocketConnectionState nextState) {
    if (_state == nextState) {
      return;
    }
    _state = nextState;
    notifyListeners();
  }

  @override
  void connect() {
    setConnectionState(SocketConnectionState.connected);
  }

  @override
  Future<void> disconnect() async {
    setConnectionState(SocketConnectionState.disconnected);
  }

  @override
  Future<void> dispose() async {
    await _controller.close();
    await super.dispose();
  }
}

class TestErrorTracker {
  final List<FlutterErrorDetails> flutterErrors = <FlutterErrorDetails>[];
  final List<Object> platformErrors = <Object>[];
  FlutterExceptionHandler? _previousFlutterErrorHandler;
  bool Function(Object, StackTrace)? _previousPlatformErrorHandler;

  void install() {
    _previousFlutterErrorHandler = FlutterError.onError;
    FlutterError.onError = (details) {
      flutterErrors.add(details);
    };
    _previousPlatformErrorHandler = PlatformDispatcher.instance.onError;
    PlatformDispatcher.instance.onError = (error, stackTrace) {
      platformErrors.add(error);
      return true;
    };
  }

  void restore() {
    FlutterError.onError = _previousFlutterErrorHandler;
    PlatformDispatcher.instance.onError = _previousPlatformErrorHandler;
  }

  void expectClean(WidgetTester tester) {
    expect(flutterErrors, isEmpty);
    expect(platformErrors, isEmpty);
    expect(tester.takeException(), isNull);
  }
}

AppSessionController createTestSessionController({
  Map<String, String> storageValues = const <String, String>{},
  AppBuildInfo buildInfo = testBuildInfo,
  List<ServicesChannelInfo> channels = const <ServicesChannelInfo>[],
  bool recorderRunning = false,
  List<DbRecording> latestVideos = const <DbRecording>[],
  List<DbRecording> bookmarkedVideos = const <DbRecording>[],
  List<DbRecording> randomVideos = const <DbRecording>[],
  List<DbJob> jobs = const <DbJob>[],
  bool workerProcessing = false,
  SocketConnectionState socketState = SocketConnectionState.connected,
  Object? loginError,
  Object? getChannelsError,
  List<FakeMediaSinkApi>? createdApis,
  List<FakeMediaSinkSocketService>? createdSockets,
}) {
  return AppSessionController(
    storage: MemorySessionStorage(storageValues),
    buildInfoLoader: (_) async => buildInfo,
    apiFactory: ({required config, token, onUnauthorized}) {
      final api = FakeMediaSinkApi(
        config: config,
        token: token,
        onUnauthorized: onUnauthorized,
        channels: channels,
        recorderRunning: recorderRunning,
        latestVideos: latestVideos,
        bookmarkedVideos: bookmarkedVideos,
        randomVideos: randomVideos,
        jobs: jobs,
        workerProcessing: workerProcessing,
        loginError: token == null ? loginError : null,
        getChannelsError: token != null ? getChannelsError : null,
      );
      createdApis?.add(api);
      return api;
    },
    socketFactory: ({required config, required token}) {
      final socket = FakeMediaSinkSocketService(
        config: config,
        token: token,
        initialState: socketState,
      );
      createdSockets?.add(socket);
      return socket;
    },
  );
}

Future<void> waitForDebounce([Duration delay = const Duration(milliseconds: 450)]) {
  return Future<void>.delayed(delay);
}

Future<void> waitForAsyncTasks([int turns = 3]) async {
  for (var index = 0; index < turns; index += 1) {
    await Future<void>.delayed(Duration.zero);
  }
}
