import "package:flutter/foundation.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../api/export.dart";
import "media_sink_api.dart";
import "models.dart";

class HistoryController extends ChangeNotifier {
  HistoryController({required MediaSinkApi api}) : _api = api {
    _ready = _load();
  }

  static const int maxEntries = 100;
  static const Duration playbackThreshold = Duration(seconds: 5);
  static const String _storageKeyPrefix = "ms_video_play_history";

  final MediaSinkApi _api;
  late final Future<void> _ready;
  bool _disposed = false;

  bool _loading = true;
  String? _error;
  List<PlayedVideoHistoryEntry> _entries = const <PlayedVideoHistoryEntry>[];

  bool get loading => _loading;
  String? get error => _error;
  List<PlayedVideoHistoryEntry> get entries => _entries;
  Future<void> get ready => _ready;

  String get _storageKey => "$_storageKeyPrefix:${Uri.encodeComponent(_api.config.origin)}";

  Future<void> recordPlayedVideo(DbRecording video) async {
    await _ready;
    if (_disposed) {
      return;
    }

    final nextEntry = PlayedVideoHistoryEntry(
      serverOrigin: _api.config.origin,
      playedAt: DateTime.now().toUtc().toIso8601String(),
      video: video,
    );

    final nextEntries = <PlayedVideoHistoryEntry>[
      nextEntry,
      ..._entries.where((entry) => entry.stableId != nextEntry.stableId),
    ];

    if (nextEntries.length > maxEntries) {
      nextEntries.removeRange(maxEntries, nextEntries.length);
    }

    _entries = nextEntries;
    _notifySafely();
    await _persist();
  }

  Future<void> removeEntry(PlayedVideoHistoryEntry entry) async {
    await _ready;
    if (_disposed) {
      return;
    }
    _entries = _entries.where((item) => item.stableId != entry.stableId).toList(growable: false);
    _notifySafely();
    await _persist();
  }

  Future<void> removeByVideo(DbRecording video) async {
    await _ready;
    if (_disposed) {
      return;
    }
    final stableId = video.recordingId?.toString() ?? video.pathRelative;
    _entries = _entries.where((entry) => entry.stableId != stableId).toList(growable: false);
    _notifySafely();
    await _persist();
  }

  Future<void> clearAll() async {
    await _ready;
    if (_disposed) {
      return;
    }
    _entries = const <PlayedVideoHistoryEntry>[];
    _notifySafely();
    final preferences = await SharedPreferences.getInstance();
    await preferences.remove(_storageKey);
  }

  Future<void> reload() async {
    if (_disposed) {
      return;
    }
    _loading = true;
    _error = null;
    _notifySafely();
    await _load();
  }

  Future<DbRecording> resolveVideo(PlayedVideoHistoryEntry entry) async {
    await _ready;
    final recordingId = entry.video.recordingId;
    if (recordingId == null) {
      return entry.video;
    }

    try {
      return await _api.getVideo(recordingId);
    } catch (error) {
      if (_looksLikeMissingVideo(error)) {
        await removeEntry(entry);
      }
      rethrow;
    }
  }

  Future<void> _load() async {
    try {
      final preferences = await SharedPreferences.getInstance();
      final raw = preferences.getString(_storageKey);
      if (raw != null && raw.isNotEmpty) {
        _entries = PlayedVideoHistoryEntry.decodeList(raw)
            .where((entry) => entry.serverOrigin.isEmpty || entry.serverOrigin == _api.config.origin)
            .toList(growable: false);
      }
    } catch (error) {
      _error = error.toString();
      _entries = const <PlayedVideoHistoryEntry>[];
    } finally {
      _loading = false;
      _notifySafely();
    }
  }

  Future<void> _persist() async {
    final preferences = await SharedPreferences.getInstance();
    if (_entries.isEmpty) {
      await preferences.remove(_storageKey);
      return;
    }
    await preferences.setString(_storageKey, PlayedVideoHistoryEntry.encodeList(_entries));
  }

  bool _looksLikeMissingVideo(Object error) {
    final message = error.toString().toLowerCase();
    return message.contains("404") || message.contains("not found") || message.contains("no rows");
  }

  void _notifySafely() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
