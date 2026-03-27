import "dart:convert";

import "../api/export.dart";

enum SessionStatus { booting, unconfigured, incompatibleVersion, loggedOut, authenticating, authenticated }

enum LibrarySection { latest, bookmarks, random }

enum JobsTab { active, completed, error, canceled }

enum SocketConnectionState { disconnected, connecting, connected, reconnecting }

enum ChannelListLayout { grid, list }

enum ChannelsSortField { recording, name, favorite, videos, size, added }

class ChannelsImportResult {
  const ChannelsImportResult({
    required this.successCount,
    required this.failureCount,
    this.cancelled = false,
  });

  final int successCount;
  final int failureCount;
  final bool cancelled;

  bool get hasFailures => failureCount > 0;
  bool get hasSuccesses => successCount > 0;
}

class PlayedVideoHistoryEntry {
  const PlayedVideoHistoryEntry({
    required this.serverOrigin,
    required this.playedAt,
    required this.video,
  });

  final String serverOrigin;
  final String playedAt;
  final DbRecording video;

  DateTime? get playedAtDateTime => DateTime.tryParse(playedAt)?.toLocal();

  String get stableId => video.recordingId?.toString() ?? video.pathRelative;

  PlayedVideoHistoryEntry copyWith({
    String? serverOrigin,
    String? playedAt,
    DbRecording? video,
  }) {
    return PlayedVideoHistoryEntry(
      serverOrigin: serverOrigin ?? this.serverOrigin,
      playedAt: playedAt ?? this.playedAt,
      video: video ?? this.video,
    );
  }

  factory PlayedVideoHistoryEntry.fromJson(Map<String, dynamic> json) {
    return PlayedVideoHistoryEntry(
      serverOrigin: json["serverOrigin"]?.toString() ?? "",
      playedAt: json["playedAt"]?.toString() ?? "",
      video: DbRecording.fromJson(Map<String, Object?>.from(json["video"] as Map)),
    );
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      "serverOrigin": serverOrigin,
      "playedAt": playedAt,
      "video": video.toJson(),
    };
  }

  static List<PlayedVideoHistoryEntry> decodeList(String raw) {
    final decoded = jsonDecode(raw);
    if (decoded is! List) {
      return const <PlayedVideoHistoryEntry>[];
    }
    return decoded
        .whereType<Map>()
        .map((entry) => PlayedVideoHistoryEntry.fromJson(Map<String, dynamic>.from(entry)))
        .toList(growable: false);
  }

  static String encodeList(List<PlayedVideoHistoryEntry> entries) {
    return jsonEncode(entries.map((entry) => entry.toJson()).toList(growable: false));
  }
}

class PlaybackProgressEntry {
  const PlaybackProgressEntry({
    required this.serverOrigin,
    required this.updatedAt,
    required this.positionSeconds,
    required this.durationSeconds,
    required this.video,
  });

  final String serverOrigin;
  final String updatedAt;
  final double positionSeconds;
  final double durationSeconds;
  final DbRecording video;

  DateTime? get updatedAtDateTime => DateTime.tryParse(updatedAt)?.toLocal();

  String get stableId => video.recordingId?.toString() ?? video.pathRelative;

  double get progressFraction {
    if (durationSeconds <= 0) {
      return 0;
    }
    return (positionSeconds / durationSeconds).clamp(0.0, 1.0);
  }

  PlaybackProgressEntry copyWith({
    String? serverOrigin,
    String? updatedAt,
    double? positionSeconds,
    double? durationSeconds,
    DbRecording? video,
  }) {
    return PlaybackProgressEntry(
      serverOrigin: serverOrigin ?? this.serverOrigin,
      updatedAt: updatedAt ?? this.updatedAt,
      positionSeconds: positionSeconds ?? this.positionSeconds,
      durationSeconds: durationSeconds ?? this.durationSeconds,
      video: video ?? this.video,
    );
  }

  factory PlaybackProgressEntry.fromJson(Map<String, dynamic> json) {
    return PlaybackProgressEntry(
      serverOrigin: json["serverOrigin"]?.toString() ?? "",
      updatedAt: json["updatedAt"]?.toString() ?? "",
      positionSeconds: (json["positionSeconds"] as num?)?.toDouble() ?? 0,
      durationSeconds: (json["durationSeconds"] as num?)?.toDouble() ?? 0,
      video: DbRecording.fromJson(Map<String, Object?>.from(json["video"] as Map)),
    );
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      "serverOrigin": serverOrigin,
      "updatedAt": updatedAt,
      "positionSeconds": positionSeconds,
      "durationSeconds": durationSeconds,
      "video": video.toJson(),
    };
  }

  static List<PlaybackProgressEntry> decodeList(String raw) {
    final decoded = jsonDecode(raw);
    if (decoded is! List) {
      return const <PlaybackProgressEntry>[];
    }
    return decoded
        .whereType<Map>()
        .map((entry) => PlaybackProgressEntry.fromJson(Map<String, dynamic>.from(entry)))
        .toList(growable: false);
  }

  static String encodeList(List<PlaybackProgressEntry> entries) {
    return jsonEncode(entries.map((entry) => entry.toJson()).toList(growable: false));
  }
}

class AppBuildInfo {
  const AppBuildInfo({required this.apiVersion, required this.version, required this.build});

  final String apiVersion;
  final String version;
  final String build;

  static AppBuildInfo fromBuildJs(String script) {
    String parseVar(String name) {
      final pattern = RegExp('$name\\s*=\\s*"([^"]*)"');
      return pattern.firstMatch(script)?.group(1) ?? "";
    }

    return AppBuildInfo(apiVersion: parseVar("APP_API_VERSION"), version: parseVar("APP_VERSION"), build: parseVar("APP_BUILD"));
  }
}

class ServerConfig {
  const ServerConfig({required this.origin, required this.apiBaseUrl, required this.fileBaseUrl, required this.socketUrl, required this.buildInfo});

  final String origin;
  final String apiBaseUrl;
  final String fileBaseUrl;
  final String socketUrl;
  final AppBuildInfo buildInfo;

  String get apiVersion => buildInfo.apiVersion;
  String get appVersion => buildInfo.version;
  String get build => buildInfo.build;

  static ServerConfig create({required String origin, required AppBuildInfo buildInfo}) {
    final normalized = _normalizeOrigin(origin);
    final uri = Uri.parse(normalized);
    final socketScheme = uri.scheme == "https" ? "wss" : "ws";
    final basePath = uri.path.endsWith("/") ? uri.path.substring(0, uri.path.length - 1) : uri.path;
    final socketUri = uri.replace(scheme: socketScheme, path: "$basePath/api/v2/ws".replaceFirst("//", "/"));

    return ServerConfig(origin: normalized, apiBaseUrl: "$normalized/api/v2", fileBaseUrl: "$normalized/videos", socketUrl: socketUri.toString(), buildInfo: buildInfo);
  }

  static String normalizeOrigin(String value) => _normalizeOrigin(value);

  static String _normalizeOrigin(String value) {
    var normalized = value.trim();
    while (normalized.endsWith("/")) {
      normalized = normalized.substring(0, normalized.length - 1);
    }
    return normalized;
  }
}

class SocketEventMessage {
  const SocketEventMessage({required this.name, required this.data});

  final String name;
  final Object? data;

  factory SocketEventMessage.fromRaw(String raw) {
    final decoded = jsonDecode(raw) as Map<String, dynamic>;
    return SocketEventMessage(
      name: decoded["name"]?.toString() ?? "",
      data: decoded["data"] is Map<String, dynamic>
          ? decoded["data"] as Map<String, dynamic>
          : decoded["data"] is Map
          ? Map<String, dynamic>.from(decoded["data"] as Map)
          : decoded["data"],
    );
  }
}
