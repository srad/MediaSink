import "dart:io";

import "package:dio/dio.dart";
import "package:path_provider/path_provider.dart";

import "../api/export.dart";
import "models.dart";

typedef UnauthorizedHandler = Future<void> Function();

class ApiVersionMismatchException implements Exception {
  const ApiVersionMismatchException(this.message);

  final String message;

  @override
  String toString() => message;
}

enum MediaSinkApiErrorKind { serverUnreachable, server, transport, unknown }

class MediaSinkApiException implements Exception {
  const MediaSinkApiException({
    required this.message,
    required this.kind,
  });

  final String message;
  final MediaSinkApiErrorKind kind;

  @override
  String toString() => message;
}

class MediaSinkApi {
  MediaSinkApi({required this.config, this.token, this.onUnauthorized})
    : _client = RestClient(
        Dio(BaseOptions(baseUrl: config.apiBaseUrl, headers: _buildHeaders(config.apiVersion, token))),
        baseUrl: config.apiBaseUrl,
      );

  final ServerConfig config;
  final String? token;
  final UnauthorizedHandler? onUnauthorized;
  final RestClient _client;

  static Map<String, String> _buildHeaders(String apiVersion, String? token) {
    return <String, String>{"X-API-Version": apiVersion, if (token?.isNotEmpty ?? false) "Authorization": "Bearer $token"};
  }

  static Future<AppBuildInfo> fetchBuildInfo(String origin) async {
    final normalized = ServerConfig.normalizeOrigin(origin);
    try {
      final response = await Dio().get<String>("$normalized/build.js", options: Options(responseType: ResponseType.plain));
      final script = response.data ?? "";
      return AppBuildInfo.fromBuildJs(script);
    } on DioException catch (error) {
      throw MediaSinkApiException(
        message: _errorMessage(error),
        kind: _errorKind(error),
      );
    }
  }

  Future<String> login({required String username, required String password}) async {
    final response = await _guardUnauthorized(
      () => _client.auth.postAuthLogin(
        authenticationRequest: RequestsAuthenticationRequest(username: username, password: password),
      ),
    );
    return response.token ?? "";
  }

  Future<List<ServicesChannelInfo>> getChannels() {
    return _guardUnauthorized(() => _client.channels.getChannels());
  }

  Future<ServicesChannelInfo> getChannel(int channelId) {
    return _guardUnauthorized(() => _client.channels.getChannelsId(id: channelId));
  }

  Future<ServicesChannelInfo> createChannel(RequestsChannelRequest request) {
    return _guardUnauthorized(() => _client.channels.postChannels(channelRequest: request));
  }

  Future<void> updateChannel(int channelId, RequestsChannelRequest request) {
    return _guardUnauthorized(() async {
      await _client.channels.patchChannelsId(id: channelId, channelRequest: request);
    });
  }

  Future<void> deleteChannel(int channelId) {
    return _guardUnauthorized(() => _client.channels.deleteChannelsId(id: channelId));
  }

  Future<void> pauseChannel(int channelId) {
    return _guardUnauthorized(() => _client.channels.postChannelsIdPause(id: channelId));
  }

  Future<void> resumeChannel(int channelId) {
    return _guardUnauthorized(() => _client.channels.postChannelsIdResume(id: channelId));
  }

  Future<void> setChannelFavorite(int channelId, bool isFavorite) {
    return _guardUnauthorized(() => isFavorite ? _client.channels.patchChannelsIdUnfav(id: channelId) : _client.channels.patchChannelsIdFav(id: channelId));
  }

  Future<void> updateChannelTags(int channelId, List<String> tags) {
    return _guardUnauthorized(
      () => _client.channels.patchChannelsIdTags(
        id: channelId,
        channelTagsUpdateRequest: RequestsChannelTagsUpdateRequest(tags: tags),
      ),
    );
  }

  Future<bool> getRecorderStatus() async {
    final response = await _guardUnauthorized(() => _client.recorder.getRecorder());
    return response.isRecording ?? false;
  }

  Future<void> pauseRecorder() {
    return _guardUnauthorized(() => _client.recorder.postRecorderPause());
  }

  Future<void> resumeRecorder() {
    return _guardUnauthorized(() => _client.recorder.postRecorderResume());
  }

  Future<List<DbRecording>> getVideos() {
    return _guardUnauthorized(() => _client.videos.getVideos());
  }

  Future<DbRecording> getVideo(int recordingId) {
    return _guardUnauthorized(() => _client.videos.getVideosId(id: recordingId));
  }

  Future<List<DbRecording>> getBookmarkedVideos() {
    return _guardUnauthorized(() => _client.videos.getVideosBookmarks());
  }

  Future<List<DbRecording>> getRandomVideos({int limit = 24}) {
    return _guardUnauthorized(() => _client.videos.getVideosRandomLimit(limit: limit));
  }

  Future<ResponsesVideoFilterResponse> filterVideos({required RequestsVideoSortColumn sortColumn, required ModelsSortOrder sortOrder, int take = 100, int skip = 0}) {
    return _guardUnauthorized(
      () => _client.videos.postVideosFilter(
        videoFilterRequest: RequestsVideoFilterRequest(sortColumn: sortColumn, sortOrder: sortOrder, take: take, skip: skip),
      ),
    );
  }

  Future<void> setVideoBookmark(int recordingId, bool isBookmarked) {
    return _guardUnauthorized(() => isBookmarked ? _client.videos.patchVideosIdUnfav(id: recordingId) : _client.videos.patchVideosIdFav(id: recordingId));
  }

  Future<void> deleteVideo(int recordingId) {
    return _guardUnauthorized(() => _client.videos.deleteVideosId(id: recordingId));
  }

  Future<void> refreshPreview(int recordingId) {
    return _guardUnauthorized(() async {
      await _client.videos.postVideosIdPreview(id: recordingId);
    });
  }

  Future<ResponsesAnalysisResponse?> getAnalysis(int recordingId) async {
    try {
      return await _client.analysis.getAnalysisId(id: recordingId);
    } on DioException catch (error) {
      final statusCode = error.response?.statusCode;
      if (statusCode == 404 || (statusCode != null && statusCode >= 500)) {
        return null;
      }
      if (statusCode == 401 && onUnauthorized != null) {
        await onUnauthorized!();
      }
      if (statusCode == 412) {
        throw ApiVersionMismatchException(error.response?.data?.toString() ?? "Client API version is incompatible with the server.");
      }
      throw MediaSinkApiException(
        message: _errorMessage(error),
        kind: _errorKind(error),
      );
    }
  }

  Future<ResponsesPreviewManifestResponse?> getPreviewManifest(int recordingId) async {
    try {
      return await _client.videos.getVideosIdPreviewManifest(id: recordingId);
    } on DioException catch (error) {
      final statusCode = error.response?.statusCode;
      if (statusCode == 404 || (statusCode != null && statusCode >= 500)) {
        return null;
      }
      if (statusCode == 401 && onUnauthorized != null) {
        await onUnauthorized!();
      }
      if (statusCode == 412) {
        throw ApiVersionMismatchException(error.response?.data?.toString() ?? "Client API version is incompatible with the server.");
      }
      throw MediaSinkApiException(
        message: _errorMessage(error),
        kind: _errorKind(error),
      );
    }
  }

  Future<ResponsesJobsResponse> getJobs({required List<DbJobStatus> states, required DbJobOrder sortOrder, int take = 100, int skip = 0}) {
    return _guardUnauthorized(
      () => _client.jobs.postJobsList(
        jobsRequest: RequestsJobsRequest(states: states, sortOrder: sortOrder, take: take, skip: skip),
      ),
    );
  }

  Future<void> deleteJob(int jobId) {
    return _guardUnauthorized(() => _client.jobs.deleteJobsId(id: jobId));
  }

  Future<bool> getWorkerStatus() async {
    final response = await _guardUnauthorized(() => _client.jobs.getJobsWorker());
    return response.isProcessing ?? false;
  }

  Future<void> pauseWorker() {
    return _guardUnauthorized(() => _client.jobs.postJobsPause());
  }

  Future<void> resumeWorker() {
    return _guardUnauthorized(() => _client.jobs.postJobsResume());
  }

  Future<ResponsesServerInfoResponse> getServerVersion() {
    return _guardUnauthorized(() => _client.admin.getAdminVersion());
  }

  Future<String> downloadVideo(DbRecording video) async {
    final directory = await getApplicationDocumentsDirectory();
    final downloadsDirectory = Directory("${directory.path}${Platform.pathSeparator}downloads");
    if (!downloadsDirectory.existsSync()) {
      downloadsDirectory.createSync(recursive: true);
    }

    final safeFilename = video.filename.replaceAll(RegExp("[\\\\/:*?\"<>|]"), "_");
    final outputPath = "${downloadsDirectory.path}${Platform.pathSeparator}$safeFilename";
    await Dio().download(videoFileUrl(video), outputPath);
    return outputPath;
  }

  String previewUrl(DbRecording video) {
    final previewPath = video.videoPreview?.previewPath;
    if (previewPath != null && previewPath.isNotEmpty) {
      return "${config.fileBaseUrl}/$previewPath/0.jpg";
    }
    return "${config.fileBaseUrl}/${video.channelName}/.previews/live.jpg";
  }

  List<String> previewFrames(DbRecording video) {
    final preview = video.videoPreview;
    final previewPath = preview?.previewPath;
    final frameCount = preview?.frameCount ?? 0;
    final frameInterval = preview?.frameInterval ?? 1;

    if (previewPath == null || previewPath.isEmpty || frameCount <= 0) {
      return const <String>[];
    }

    const maxPreviewFrames = 400;
    final sampleEveryFrames = (frameCount / maxPreviewFrames).ceil().clamp(1, frameCount);
    final frames = <String>[];

    for (var frameIndex = 0; frameIndex < frameCount; frameIndex += sampleEveryFrames) {
      final timestamp = frameIndex * frameInterval;
      frames.add("${config.fileBaseUrl}/$previewPath/$timestamp.jpg");
    }

    final lastFrameUrl = "${config.fileBaseUrl}/$previewPath/${(frameCount - 1) * frameInterval}.jpg";
    if (frames.isEmpty || frames.last != lastFrameUrl) {
      frames.add(lastFrameUrl);
    }

    return frames;
  }

  String videoFileUrl(DbRecording video) => "${config.fileBaseUrl}/${video.pathRelative}";

  Future<T> _guardUnauthorized<T>(Future<T> Function() action) async {
    try {
      return await action();
    } on DioException catch (error) {
      final statusCode = error.response?.statusCode;
      if (statusCode == 401 && onUnauthorized != null) {
        await onUnauthorized!();
      }
      if (statusCode == 412) {
        throw ApiVersionMismatchException(error.response?.data?.toString() ?? "Client API version is incompatible with the server.");
      }
      throw MediaSinkApiException(
        message: _errorMessage(error),
        kind: _errorKind(error),
      );
    }
  }

  static MediaSinkApiErrorKind _errorKind(DioException error) {
    switch (error.type) {
      case DioExceptionType.connectionError:
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.receiveTimeout:
      case DioExceptionType.sendTimeout:
        return MediaSinkApiErrorKind.serverUnreachable;
      case DioExceptionType.badResponse:
        return MediaSinkApiErrorKind.server;
      case DioExceptionType.cancel:
      case DioExceptionType.badCertificate:
        return MediaSinkApiErrorKind.transport;
      case DioExceptionType.unknown:
        if (error.error is HttpException || error.error is SocketException) {
          return MediaSinkApiErrorKind.serverUnreachable;
        }
        return MediaSinkApiErrorKind.unknown;
    }
  }

  static String _errorMessage(DioException error) {
    final data = error.response?.data;
    if (data is Map<String, dynamic>) {
      if (data["error"] != null) {
        return data["error"].toString();
      }
      if (data["message"] != null) {
        return data["message"].toString();
      }
    }
    final nestedError = error.error?.toString();
    if (nestedError != null && nestedError.isNotEmpty && nestedError != "null") {
      if (nestedError.toLowerCase().contains("connection reset by peer")) {
        return "The server connection was interrupted. Try again.";
      }
      return nestedError;
    }
    final message = error.message;
    if (message != null && message.isNotEmpty && message != "null") {
      if (message.toLowerCase().contains("connection reset by peer")) {
        return "The server connection was interrupted. Try again.";
      }
      return message;
    }
    if (_errorKind(error) == MediaSinkApiErrorKind.serverUnreachable) {
      return "Server unreachable. Check the connection and try again.";
    }
    return "Request failed.";
  }
}
