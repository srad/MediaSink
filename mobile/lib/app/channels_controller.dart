import "dart:async";
import "dart:convert";
import "dart:io";

import "package:file_selector/file_selector.dart";
import "package:flutter/foundation.dart";
import "package:path_provider/path_provider.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../api/export.dart";
import "error_utils.dart";
import "media_sink_api.dart";
import "models.dart";
import "socket_service.dart";

class ChannelsController extends ChangeNotifier {
  ChannelsController({required MediaSinkApi api, MediaSinkSocketService? socket}) : _api = api {
    _subscription = socket?.events.listen(_handleSocketEvent);
    _ready = _initialize();
  }

  static const String _streamSearchKey = "ms_stream_search";
  static const String _streamFavoritesOnlyKey = "ms_stream_favorites_only";
  static const String _streamTabIndexKey = "ms_stream_tab_index";
  static const String _channelsSearchKey = "ms_channels_search";
  static const String _channelsFavoritesOnlyKey = "ms_channels_favorites_only";
  static const String _channelsLayoutKey = "ms_channels_layout";
  static const String _channelsSortFieldKey = "ms_channels_sort_field";
  static const String _channelsSortDescendingKey = "ms_channels_sort_desc";

  final MediaSinkApi _api;
  StreamSubscription? _subscription;
  Timer? _refreshDebounce;
  late final Future<void> _ready;
  bool _disposed = false;

  bool _loading = false;
  String? _error;
  bool _isRecorderRunning = false;
  bool _isImportExportBusy = false;
  List<ServicesChannelInfo> _channels = const <ServicesChannelInfo>[];
  String _streamSearchQuery = "";
  bool _streamFavoritesOnly = false;
  int _streamTabIndex = 0;
  String _channelsSearchQuery = "";
  bool _channelsFavoritesOnly = false;
  ChannelListLayout _channelsLayout = ChannelListLayout.grid;
  ChannelsSortField _channelsSortField = ChannelsSortField.recording;
  bool _channelsSortDescending = true;

  bool get loading => _loading;
  String? get error => _error;
  bool get isRecorderRunning => _isRecorderRunning;
  bool get isImportExportBusy => _isImportExportBusy;
  List<ServicesChannelInfo> get channels => _channels;
  String get streamSearchQuery => _streamSearchQuery;
  bool get streamFavoritesOnly => _streamFavoritesOnly;
  int get streamTabIndex => _streamTabIndex;
  String get channelsSearchQuery => _channelsSearchQuery;
  bool get channelsFavoritesOnly => _channelsFavoritesOnly;
  ChannelListLayout get channelsLayout => _channelsLayout;
  ChannelsSortField get channelsSortField => _channelsSortField;
  bool get channelsSortDescending => _channelsSortDescending;
  Future<void> get ready => _ready;

  List<ServicesChannelInfo> get streamOnlineChannels =>
      _filterChannels(_channels.where((channel) => (channel.isOnline ?? false) && (channel.isPaused ?? false) != true), sortForChannelsView: false);
  List<ServicesChannelInfo> get streamOfflineChannels =>
      _filterChannels(_channels.where((channel) => (channel.isOnline ?? false) == false && (channel.isPaused ?? false) != true), sortForChannelsView: false);
  List<ServicesChannelInfo> get streamDisabledChannels => _filterChannels(_channels.where((channel) => channel.isPaused ?? false), sortForChannelsView: false);
  List<ServicesChannelInfo> get filteredChannels => _filterChannels(_channels, query: _channelsSearchQuery, favoritesOnly: _channelsFavoritesOnly, sortForChannelsView: true);

  ServicesChannelInfo? channelById(int id) {
    for (final channel in _channels) {
      if (channel.channelId == id) {
        return channel;
      }
    }
    return null;
  }

  Future<void> _initialize() async {
    await _loadPreferences();
    await refresh();
  }

  Future<void> refresh() async {
    if (_disposed) {
      return;
    }
    _loading = true;
    _error = null;
    _notifySafely();

    try {
      final results = await Future.wait<dynamic>(<Future<dynamic>>[_api.getChannels(), _api.getRecorderStatus()]);
      if (_disposed) {
        return;
      }
      _channels = _sortChannelsByName(results[0] as List<ServicesChannelInfo>);
      _isRecorderRunning = results[1] as bool;
    } catch (error) {
      if (_disposed) {
        return;
      }
      _error = friendlyErrorMessage(error);
    } finally {
      if (!_disposed) {
        _loading = false;
        _notifySafely();
      }
    }
  }

  Future<ServicesChannelInfo> loadChannel(int id) {
    return _api.getChannel(id);
  }

  Future<void> saveChannel({int? id, required RequestsChannelRequest request}) async {
    if (id == null) {
      await _api.createChannel(request);
    } else {
      await _api.updateChannel(id, request);
    }
    await refresh();
  }

  Future<void> deleteChannel(int id) async {
    await _api.deleteChannel(id);
    _channels = _channels.where((channel) => channel.channelId != id).toList(growable: false);
    _notifySafely();
    await refresh();
  }

  Future<void> toggleFavorite(ServicesChannelInfo channel) async {
    await _api.setChannelFavorite(channel.channelId!, channel.fav ?? false);
    await refresh();
  }

  Future<void> togglePause(ServicesChannelInfo channel) async {
    if (channel.isPaused ?? false) {
      await _api.resumeChannel(channel.channelId!);
    } else {
      await _api.pauseChannel(channel.channelId!);
    }
    await refresh();
  }

  Future<void> updateTags(int id, List<String> tags) async {
    await _api.updateChannelTags(id, tags);
    await refresh();
  }

  Future<void> toggleRecorder() async {
    if (_isRecorderRunning) {
      await _api.pauseRecorder();
    } else {
      await _api.resumeRecorder();
    }
    _isRecorderRunning = !_isRecorderRunning;
    _notifySafely();
  }

  void setStreamSearchQuery(String value) {
    final normalized = value.trimLeft();
    if (_streamSearchQuery == normalized) {
      return;
    }
    _streamSearchQuery = normalized;
    _notifySafely();
    unawaited(_saveStringPreference(_streamSearchKey, normalized));
  }

  void setStreamFavoritesOnly(bool value) {
    if (_streamFavoritesOnly == value) {
      return;
    }
    _streamFavoritesOnly = value;
    _notifySafely();
    unawaited(_saveBoolPreference(_streamFavoritesOnlyKey, value));
  }

  void setStreamTabIndex(int value) {
    final normalized = value.clamp(0, 2);
    if (_streamTabIndex == normalized) {
      return;
    }
    _streamTabIndex = normalized;
    _notifySafely();
    unawaited(_saveIntPreference(_streamTabIndexKey, normalized));
  }

  void setChannelsSearchQuery(String value) {
    final normalized = value.trimLeft();
    if (_channelsSearchQuery == normalized) {
      return;
    }
    _channelsSearchQuery = normalized;
    _notifySafely();
    unawaited(_saveStringPreference(_channelsSearchKey, normalized));
  }

  void setChannelsFavoritesOnly(bool value) {
    if (_channelsFavoritesOnly == value) {
      return;
    }
    _channelsFavoritesOnly = value;
    _notifySafely();
    unawaited(_saveBoolPreference(_channelsFavoritesOnlyKey, value));
  }

  void setChannelsLayout(ChannelListLayout value) {
    if (_channelsLayout == value) {
      return;
    }
    _channelsLayout = value;
    _notifySafely();
    unawaited(_saveStringPreference(_channelsLayoutKey, value.name));
  }

  void setChannelsSortField(ChannelsSortField value) {
    if (_channelsSortField == value) {
      return;
    }
    _channelsSortField = value;
    _notifySafely();
    unawaited(_saveStringPreference(_channelsSortFieldKey, value.name));
  }

  void setChannelsSortDescending(bool value) {
    if (_channelsSortDescending == value) {
      return;
    }
    _channelsSortDescending = value;
    _notifySafely();
    unawaited(_saveBoolPreference(_channelsSortDescendingKey, value));
  }

  Future<String?> exportChannelsJson() async {
    await _ready;
    if (_isImportExportBusy) {
      return null;
    }

    _isImportExportBusy = true;
    _notifySafely();

    try {
      final suggestedName = "mediasink-channels-${DateTime.now().toUtc().toIso8601String().replaceAll(":", "-")}.json";
      final saveLocation = await getSaveLocation(
        acceptedTypeGroups: const <XTypeGroup>[XTypeGroup(label: "JSON", extensions: <String>["json"])],
        suggestedName: suggestedName,
        confirmButtonText: "Export",
      );

      final outputPath = saveLocation?.path ?? await _fallbackExportPath(suggestedName);
      final file = File(outputPath);
      await file.writeAsString(
        jsonEncode(
          _channels.map((channel) => channel.toJson()).toList(growable: false),
        ),
      );
      return outputPath;
    } finally {
      if (!_disposed) {
        _isImportExportBusy = false;
        _notifySafely();
      }
    }
  }

  Future<ChannelsImportResult> importChannelsFromFile() async {
    await _ready;
    if (_isImportExportBusy) {
      return const ChannelsImportResult(successCount: 0, failureCount: 0, cancelled: true);
    }

    _isImportExportBusy = true;
    _notifySafely();

    try {
      final file = await openFile(
        acceptedTypeGroups: const <XTypeGroup>[XTypeGroup(label: "JSON", extensions: <String>["json"])],
        confirmButtonText: "Import",
      );

      if (file == null) {
        return const ChannelsImportResult(successCount: 0, failureCount: 0, cancelled: true);
      }

      final raw = await file.readAsString();
      final decoded = jsonDecode(raw);
      if (decoded is! List) {
        throw const FormatException("The selected file does not contain a JSON array.");
      }

      var successCount = 0;
      var failureCount = 0;
      for (final item in decoded) {
        if (item is! Map) {
          failureCount += 1;
          continue;
        }

        final request = _channelRequestFromJson(Map<String, dynamic>.from(item));
        try {
          await _api.createChannel(request);
          successCount += 1;
        } catch (_) {
          failureCount += 1;
        }
      }

      await refresh();
      return ChannelsImportResult(successCount: successCount, failureCount: failureCount);
    } finally {
      if (!_disposed) {
        _isImportExportBusy = false;
        _notifySafely();
      }
    }
  }

  void _handleSocketEvent(dynamic event) {
    if (_disposed || event is! SocketEventMessage) {
      return;
    }

    switch (event.name) {
      case "channel:online":
        _patchChannelState(_socketInt(event.data), isOnline: true);
        return;
      case "channel:offline":
        _patchChannelState(_socketInt(event.data), isOnline: false, isRecording: false);
        return;
      case "channel:start":
        _patchChannelState(_socketInt(event.data), isOnline: true, isRecording: true);
        return;
      case "channel:thumbnail":
        _bumpChannelPreview(_socketInt(event.data));
        return;
      case "recording:add":
        final recording = _socketRecording(event.data);
        if (recording == null) {
          _scheduleRefresh();
          return;
        }
        _incrementChannelRecordingStats(recording);
        return;
      default:
        if (event.name.startsWith("channel:") || event.name.startsWith("job:preview:")) {
          _scheduleRefresh();
        }
        return;
    }
  }

  void _patchChannelState(
    int? channelId, {
    bool? isOnline,
    bool? isRecording,
  }) {
    if (channelId == null) {
      return;
    }
    var changed = false;
    _channels = _channels.map((channel) {
      if (channel.channelId != channelId) {
        return channel;
      }
      changed = true;
      return _channelWith(
        channel,
        isOnline: isOnline,
        isRecording: isRecording,
        isPaused: isRecording == true ? false : channel.isPaused,
      );
    }).toList(growable: false);
    if (changed) {
      _notifySafely();
    }
  }

  void _bumpChannelPreview(int? channelId) {
    if (channelId == null) {
      return;
    }
    var changed = false;
    _channels = _channels.map((channel) {
      if (channel.channelId != channelId) {
        return channel;
      }
      changed = true;
      final preview = channel.preview;
      if (preview == null || preview.isEmpty) {
        return channel;
      }
      final previewBase = preview.split("?").first;
      return _channelWith(channel, preview: "$previewBase?t=${DateTime.now().millisecondsSinceEpoch}");
    }).toList(growable: false);
    if (changed) {
      _notifySafely();
    }
  }

  void _incrementChannelRecordingStats(DbRecording recording) {
    var changed = false;
    _channels = _channels.map((channel) {
      if (channel.channelId != recording.channelId) {
        return channel;
      }
      changed = true;
      final nextRecordings = <DbRecording>[
        recording,
        ...(channel.recordings ?? const <DbRecording>[]),
      ];
      return _channelWith(
        channel,
        recordings: nextRecordings,
        recordingsCount: (channel.recordingsCount ?? (channel.recordings?.length ?? 0)) + 1,
        recordingsSize: (channel.recordingsSize ?? 0) + (recording.size ?? 0),
      );
    }).toList(growable: false);
    if (changed) {
      _notifySafely();
    } else {
      _scheduleRefresh();
    }
  }

  void _scheduleRefresh() {
    _refreshDebounce?.cancel();
    _refreshDebounce = Timer(const Duration(milliseconds: 350), refresh);
  }

  List<ServicesChannelInfo> _filterChannels(
    Iterable<ServicesChannelInfo> source, {
    String? query,
    bool? favoritesOnly,
    required bool sortForChannelsView,
  }) {
    final normalizedQuery = (query ?? _streamSearchQuery).trim().toLowerCase();
    final showFavoritesOnly = favoritesOnly ?? _streamFavoritesOnly;
    final terms = normalizedQuery.split(RegExp("\\s+")).where((part) => part.isNotEmpty).toList(growable: false);
    final tagTerms = terms.where((term) => term.startsWith("#")).map((term) => term.substring(1)).where((term) => term.isNotEmpty).toList(growable: false);
    final textTerms = terms.where((term) => !term.startsWith("#")).toList(growable: false);

    final filtered = source.where((channel) {
      if (showFavoritesOnly && !(channel.fav ?? false)) {
        return false;
      }

      final channelName = (channel.channelName ?? "").toLowerCase();
      final displayName = (channel.displayName ?? "").toLowerCase();
      final tags = (channel.tags ?? const <String>[]).map((tag) => tag.toLowerCase()).toList(growable: false);

      if (textTerms.isNotEmpty &&
          !textTerms.every((term) => channelName.contains(term) || displayName.contains(term))) {
        return false;
      }

      if (tagTerms.isNotEmpty && !tagTerms.every((term) => tags.any((tag) => tag.contains(term)))) {
        return false;
      }

      return true;
    }).toList(growable: false);

    return sortForChannelsView
        ? _sortChannelsForChannelsView(filtered)
        : _sortChannelsByName(filtered);
  }

  List<ServicesChannelInfo> _sortChannelsByName(List<ServicesChannelInfo> items) {
    final sorted = items.toList(growable: false);
    sorted.sort((left, right) {
      final leftName = (left.displayName ?? left.channelName ?? "").toLowerCase();
      final rightName = (right.displayName ?? right.channelName ?? "").toLowerCase();
      return leftName.compareTo(rightName);
    });
    return sorted;
  }

  List<ServicesChannelInfo> _sortChannelsForChannelsView(List<ServicesChannelInfo> items) {
    final sorted = items.toList(growable: false);
    sorted.sort((left, right) {
      final compare = switch (_channelsSortField) {
        ChannelsSortField.recording => _compareBool(left.isRecording ?? false, right.isRecording ?? false),
        ChannelsSortField.name => _compareText(left.displayName ?? left.channelName ?? "", right.displayName ?? right.channelName ?? ""),
        ChannelsSortField.favorite => _compareBool(left.fav ?? false, right.fav ?? false),
        ChannelsSortField.videos => _compareInt(left.recordingsCount ?? 0, right.recordingsCount ?? 0),
        ChannelsSortField.size => _compareInt(left.recordingsSize ?? 0, right.recordingsSize ?? 0),
        ChannelsSortField.added => _compareInt(_channelCreatedAtMillis(left), _channelCreatedAtMillis(right)),
      };
      if (compare != 0) {
        return _channelsSortDescending ? -compare : compare;
      }
      return _compareText(left.displayName ?? left.channelName ?? "", right.displayName ?? right.channelName ?? "");
    });
    return sorted;
  }

  int _compareBool(bool left, bool right) => (left ? 1 : 0).compareTo(right ? 1 : 0);

  int _compareInt(int left, int right) => left.compareTo(right);

  int _compareText(String left, String right) => left.toLowerCase().compareTo(right.toLowerCase());

  int _channelCreatedAtMillis(ServicesChannelInfo channel) {
    return DateTime.tryParse(channel.createdAt ?? "")?.millisecondsSinceEpoch ?? 0;
  }

  ServicesChannelInfo _channelWith(
    ServicesChannelInfo channel, {
    bool? isOnline,
    bool? isPaused,
    bool? isRecording,
    String? preview,
    List<DbRecording>? recordings,
    int? recordingsCount,
    int? recordingsSize,
  }) {
    final json = channel.toJson();
    if (isOnline != null) {
      json["isOnline"] = isOnline;
    }
    if (isPaused != null) {
      json["isPaused"] = isPaused;
    }
    if (isRecording != null) {
      json["isRecording"] = isRecording;
    }
    if (preview != null) {
      json["preview"] = preview;
    }
    if (recordings != null) {
      json["recordings"] = recordings.map((recording) => recording.toJson()).toList(growable: false);
    }
    if (recordingsCount != null) {
      json["recordingsCount"] = recordingsCount;
    }
    if (recordingsSize != null) {
      json["recordingsSize"] = recordingsSize;
    }
    return ServicesChannelInfo.fromJson(json);
  }

  DbRecording? _socketRecording(Object? payload) {
    if (payload is Map<String, dynamic>) {
      return DbRecording.fromJson(Map<String, Object?>.from(payload));
    }
    if (payload is Map) {
      return DbRecording.fromJson(Map<String, Object?>.from(payload));
    }
    return null;
  }

  int? _socketInt(Object? payload) {
    if (payload is int) {
      return payload;
    }
    if (payload is num) {
      return payload.toInt();
    }
    if (payload is String) {
      return int.tryParse(payload);
    }
    return null;
  }

  RequestsChannelRequest _channelRequestFromJson(Map<String, dynamic> json) {
    final rawTags = json["tags"];
    final tags = rawTags is List ? rawTags.map((tag) => tag.toString()).where((tag) => tag.isNotEmpty).toList(growable: false) : null;
    return RequestsChannelRequest(
      channelName: json["channelName"]?.toString() ?? "",
      displayName: json["displayName"]?.toString() ?? "",
      url: json["url"]?.toString() ?? "",
      minDuration: (json["minDuration"] as num?)?.toInt() ?? 20,
      skipStart: (json["skipStart"] as num?)?.toInt() ?? 0,
      isPaused: json["isPaused"] == true,
      tags: tags,
    );
  }

  Future<void> _loadPreferences() async {
    final preferences = await SharedPreferences.getInstance();
    _streamSearchQuery = preferences.getString(_streamSearchKey) ?? "";
    _streamFavoritesOnly = preferences.getBool(_streamFavoritesOnlyKey) ?? false;
    _streamTabIndex = (preferences.getInt(_streamTabIndexKey) ?? 0).clamp(0, 2);
    _channelsSearchQuery = preferences.getString(_channelsSearchKey) ?? "";
    _channelsFavoritesOnly = preferences.getBool(_channelsFavoritesOnlyKey) ?? false;
    _channelsLayout = ChannelListLayout.values.firstWhere(
      (layout) => layout.name == (preferences.getString(_channelsLayoutKey) ?? ChannelListLayout.grid.name),
      orElse: () => ChannelListLayout.grid,
    );
    _channelsSortField = ChannelsSortField.values.firstWhere(
      (field) => field.name == (preferences.getString(_channelsSortFieldKey) ?? ChannelsSortField.recording.name),
      orElse: () => ChannelsSortField.recording,
    );
    _channelsSortDescending = preferences.getBool(_channelsSortDescendingKey) ?? true;
  }

  Future<void> _saveStringPreference(String key, String value) async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(key, value);
  }

  Future<void> _saveBoolPreference(String key, bool value) async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setBool(key, value);
  }

  Future<void> _saveIntPreference(String key, int value) async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setInt(key, value);
  }

  Future<String> _fallbackExportPath(String suggestedName) async {
    final directory = await getApplicationDocumentsDirectory();
    final exportsDirectory = Directory("${directory.path}${Platform.pathSeparator}exports");
    if (!exportsDirectory.existsSync()) {
      exportsDirectory.createSync(recursive: true);
    }
    return "${exportsDirectory.path}${Platform.pathSeparator}$suggestedName";
  }

  void _notifySafely() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _refreshDebounce?.cancel();
    _subscription?.cancel();
    super.dispose();
  }
}
