import "package:flutter/foundation.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../api/export.dart";
import "media_sink_api.dart";
import "models.dart";

class PlaybackProgressController extends ChangeNotifier {
  PlaybackProgressController({required MediaSinkApi api}) : _api = api {
    _ready = _load();
  }

  static const int maxEntries = 100;
  static const Duration minimumResumePosition = Duration(seconds: 5);
  static const double completionThreshold = 0.95;
  static const String _storageKeyPrefix = "ms_playback_progress";

  final MediaSinkApi _api;
  late final Future<void> _ready;
  bool _disposed = false;
  List<PlaybackProgressEntry> _entries = const <PlaybackProgressEntry>[];

  List<PlaybackProgressEntry> get entries => _entries;
  Future<void> get ready => _ready;

  String get _storageKey => "$_storageKeyPrefix:${Uri.encodeComponent(_api.config.origin)}";

  PlaybackProgressEntry? entryForVideo(DbRecording video) {
    final stableId = video.recordingId?.toString() ?? video.pathRelative;
    for (final entry in _entries) {
      if (entry.stableId == stableId) {
        return entry;
      }
    }
    return null;
  }

  Future<void> recordProgress({
    required DbRecording video,
    required double positionSeconds,
    required double durationSeconds,
  }) async {
    await _ready;
    if (_disposed) {
      return;
    }

    if (durationSeconds <= 0 || positionSeconds < minimumResumePosition.inSeconds) {
      await removeByVideo(video);
      return;
    }

    final progressFraction = durationSeconds <= 0 ? 0 : (positionSeconds / durationSeconds);
    if (progressFraction >= completionThreshold) {
      await removeByVideo(video);
      return;
    }

    final nextEntry = PlaybackProgressEntry(
      serverOrigin: _api.config.origin,
      updatedAt: DateTime.now().toUtc().toIso8601String(),
      positionSeconds: positionSeconds,
      durationSeconds: durationSeconds,
      video: video,
    );

    final nextEntries = <PlaybackProgressEntry>[
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

  Future<void> removeByVideo(DbRecording video) async {
    await _ready;
    if (_disposed) {
      return;
    }

    final stableId = video.recordingId?.toString() ?? video.pathRelative;
    final nextEntries = _entries.where((entry) => entry.stableId != stableId).toList(growable: false);
    if (identical(nextEntries, _entries) || nextEntries.length == _entries.length) {
      return;
    }
    _entries = nextEntries;
    _notifySafely();
    await _persist();
  }

  Future<void> _load() async {
    final preferences = await SharedPreferences.getInstance();
    final raw = preferences.getString(_storageKey);
    if (raw == null || raw.isEmpty) {
      _entries = const <PlaybackProgressEntry>[];
      return;
    }
    _entries = PlaybackProgressEntry.decodeList(raw)
        .where((entry) => entry.serverOrigin.isEmpty || entry.serverOrigin == _api.config.origin)
        .toList(growable: false);
  }

  Future<void> _persist() async {
    final preferences = await SharedPreferences.getInstance();
    if (_entries.isEmpty) {
      await preferences.remove(_storageKey);
      return;
    }
    await preferences.setString(_storageKey, PlaybackProgressEntry.encodeList(_entries));
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
