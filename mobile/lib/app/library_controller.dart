import "dart:async";
import "dart:math" as math;

import "package:flutter/foundation.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../api/export.dart";
import "error_utils.dart";
import "media_sink_api.dart";
import "models.dart";
import "socket_service.dart";

class LibraryController extends ChangeNotifier {
  LibraryController({required MediaSinkApi api, MediaSinkSocketService? socket}) : _api = api {
    _subscription = socket?.events.listen(_handleSocketEvent);
    _ready = _initialize();
  }

  static const int defaultPageSize = 100;
  static const int defaultRandomLimit = 25;
  static const List<int> pageSizeOptions = <int>[25, 50, 100, 200, 500, 1000];
  static const String _sectionKey = "ms_library_section";
  static const String _sortColumnKey = "ms_library_sort_column";
  static const String _sortOrderKey = "ms_library_sort_order";
  static const String _pageSizeKey = "ms_library_page_size";
  static const String _skipKey = "ms_library_skip";
  static const String _randomLimitKey = "ms_library_random_limit";
  static const String _bookmarkChannelFilterKey = "ms_library_bookmark_channel_filter";

  final MediaSinkApi _api;
  StreamSubscription? _subscription;
  Timer? _refreshDebounce;
  late final Future<void> _ready;
  bool _disposed = false;
  int _refreshGeneration = 0;

  bool _loading = false;
  String? _error;
  LibrarySection _section = LibrarySection.latest;
  RequestsVideoSortColumn _sortColumn = RequestsVideoSortColumn.createdAt;
  ModelsSortOrder _sortOrder = ModelsSortOrder.desc;
  int _pageSize = defaultPageSize;
  int _skip = 0;
  int _totalCount = 0;
  int _randomLimit = defaultRandomLimit;
  String _bookmarkChannelFilter = "";
  List<DbRecording> _videos = const <DbRecording>[];
  List<DbRecording> _bookmarkedVideos = const <DbRecording>[];

  bool get loading => _loading;
  String? get error => _error;
  LibrarySection get section => _section;
  RequestsVideoSortColumn get sortColumn => _sortColumn;
  ModelsSortOrder get sortOrder => _sortOrder;
  int get pageSize => _pageSize;
  int get skip => _skip;
  int get totalCount => _section == LibrarySection.latest ? _totalCount : videos.length;
  int get currentPage => _section == LibrarySection.latest ? (_skip ~/ _pageSize) + 1 : 1;
  int get totalPages {
    final count = totalCount;
    if (count <= 0) {
      return 1;
    }
    final size = _section == LibrarySection.latest ? _pageSize : count;
    return math.max(1, (count / math.max(size, 1)).ceil());
  }

  int get visibleStart {
    if (videos.isEmpty) {
      return 0;
    }
    if (_section == LibrarySection.latest) {
      return _skip + 1;
    }
    return 1;
  }

  int get visibleEnd {
    if (videos.isEmpty) {
      return 0;
    }
    if (_section == LibrarySection.latest) {
      return math.min(_skip + _videos.length, _totalCount);
    }
    return videos.length;
  }

  bool get canGoToPreviousPage => _section == LibrarySection.latest && _skip > 0;
  bool get canGoToNextPage => _section == LibrarySection.latest && visibleEnd < _totalCount;
  int get randomLimit => _randomLimit;
  String get bookmarkChannelFilter => _bookmarkChannelFilter;
  Future<void> get ready => _ready;

  List<String> get bookmarkChannels {
    final values = _bookmarkedVideos.map((video) => video.channelName).toSet().toList(growable: false);
    final sorted = values.toList(growable: false);
    sorted.sort((left, right) => left.toLowerCase().compareTo(right.toLowerCase()));
    return sorted;
  }

  List<int?> get visiblePageItems {
    final total = totalPages;
    if (_section != LibrarySection.latest || total <= 1) {
      return const <int?>[];
    }
    if (total <= 5) {
      return List<int?>.generate(total, (index) => index + 1);
    }
    if (currentPage <= 2) {
      return <int?>[1, 2, 3, null, total];
    }
    if (currentPage >= total - 1) {
      return <int?>[1, null, total - 2, total - 1, total];
    }
    return <int?>[1, null, currentPage, null, total];
  }

  List<DbRecording> get videos {
    if (_section != LibrarySection.bookmarks) {
      return _videos;
    }
    if (_bookmarkChannelFilter.isEmpty) {
      return _bookmarkedVideos;
    }
    return _bookmarkedVideos.where((video) => video.channelName == _bookmarkChannelFilter).toList(growable: false);
  }

  Future<void> _initialize() async {
    await _loadPreferences();
    await refresh();
  }

  Future<void> setSection(LibrarySection section) async {
    if (_section == section) {
      return;
    }
    _section = section;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> setSortColumn(RequestsVideoSortColumn sortColumn) async {
    if (_sortColumn == sortColumn) {
      return;
    }
    _sortColumn = sortColumn;
    _skip = 0;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> setSortOrder(ModelsSortOrder sortOrder) async {
    if (_sortOrder == sortOrder) {
      return;
    }
    _sortOrder = sortOrder;
    _skip = 0;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> setPageSize(int pageSize) async {
    if (_pageSize == pageSize) {
      return;
    }
    _pageSize = pageSize;
    _skip = 0;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> applyLatestFilters({required RequestsVideoSortColumn sortColumn, required ModelsSortOrder sortOrder, required int pageSize}) async {
    final changed = _sortColumn != sortColumn || _sortOrder != sortOrder || _pageSize != pageSize || _skip != 0;
    _sortColumn = sortColumn;
    _sortOrder = sortOrder;
    _pageSize = pageSize;
    _skip = 0;
    if (!changed) {
      return;
    }
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> resetLatestFilters() async {
    _sortColumn = RequestsVideoSortColumn.createdAt;
    _sortOrder = ModelsSortOrder.desc;
    _pageSize = defaultPageSize;
    _skip = 0;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> goToPage(int page) async {
    if (_section != LibrarySection.latest) {
      return;
    }
    final targetPage = page.clamp(1, totalPages);
    final targetSkip = (targetPage - 1) * _pageSize;
    if (targetSkip == _skip) {
      return;
    }
    _skip = targetSkip;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> nextPage() => goToPage(currentPage + 1);

  Future<void> previousPage() => goToPage(currentPage - 1);

  Future<void> setRandomLimit(int limit) async {
    if (_randomLimit == limit) {
      return;
    }
    _randomLimit = limit;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  void setBookmarkChannelFilter(String? channelName) {
    final nextValue = channelName ?? "";
    if (_bookmarkChannelFilter == nextValue) {
      return;
    }
    _bookmarkChannelFilter = nextValue;
    _notifySafely();
    unawaited(_savePreferences());
  }

  Future<void> refresh() async {
    if (_disposed) {
      return;
    }
    final generation = ++_refreshGeneration;
    _loading = true;
    _error = null;
    _notifySafely();

    try {
      if (_section == LibrarySection.latest) {
        await _refreshLatest(generation);
      } else if (_section == LibrarySection.bookmarks) {
        final bookmarkedVideos = await _api.getBookmarkedVideos();
        if (_isStale(generation)) {
          return;
        }
        _bookmarkedVideos = bookmarkedVideos;
        if (_bookmarkChannelFilter.isNotEmpty && !_bookmarkedVideos.any((video) => video.channelName == _bookmarkChannelFilter)) {
          _bookmarkChannelFilter = "";
          await _savePreferences();
        }
      } else {
        final randomVideos = await _api.getRandomVideos(limit: _randomLimit);
        if (_isStale(generation)) {
          return;
        }
        _videos = randomVideos;
      }
    } catch (error) {
      if (_isStale(generation)) {
        return;
      }
      _error = friendlyErrorMessage(error);
    } finally {
      if (!_disposed && generation == _refreshGeneration) {
        _loading = false;
        _notifySafely();
      }
    }
  }

  Future<void> toggleBookmark(DbRecording video) async {
    await _api.setVideoBookmark(video.recordingId!, video.bookmark ?? false);
    final nextBookmark = !(video.bookmark ?? false);
    _videos = _videos.map((item) => _videoWithBookmark(item, target: video, bookmark: nextBookmark)).toList(growable: false);
    if (nextBookmark) {
      _bookmarkedVideos = _bookmarkedVideos.any((item) => item.recordingId == video.recordingId)
          ? _bookmarkedVideos.map((item) => _videoWithBookmark(item, target: video, bookmark: true)).toList(growable: false)
          : <DbRecording>[_videoWithBookmark(video, target: video, bookmark: true), ..._bookmarkedVideos];
    } else {
      _bookmarkedVideos = _bookmarkedVideos.where((item) => item.recordingId != video.recordingId).toList(growable: false);
    }
    if (_bookmarkChannelFilter.isNotEmpty && !_bookmarkedVideos.any((item) => item.channelName == _bookmarkChannelFilter)) {
      _bookmarkChannelFilter = "";
      await _savePreferences();
    }
    _notifySafely();
  }

  Future<void> deleteVideo(DbRecording video) async {
    await _api.deleteVideo(video.recordingId!);
    _videos = _videos.where((item) => item.recordingId != video.recordingId).toList(growable: false);
    _bookmarkedVideos = _bookmarkedVideos.where((item) => item.recordingId != video.recordingId).toList(growable: false);
    if (_section == LibrarySection.latest) {
      _totalCount = math.max(_totalCount - 1, 0);
      if (_videos.isEmpty && _skip > 0) {
        _skip = math.max(_skip - _pageSize, 0);
        await _savePreferences();
        await refresh();
        return;
      }
    }
    _notifySafely();
  }

  Future<void> refreshPreview(DbRecording video) async {
    await _api.refreshPreview(video.recordingId!);
    _scheduleRefresh();
  }

  Future<void> _refreshLatest(int generation) async {
    var response = await _api.filterVideos(sortColumn: _sortColumn, sortOrder: _sortOrder, take: _pageSize, skip: _skip);
    if (_isStale(generation)) {
      return;
    }

    var nextSkip = response.skip ?? _skip;
    var nextTake = response.take ?? _pageSize;
    var nextTotalCount = response.totalCount ?? ((response.videos ?? const <DbRecording>[]).length + nextSkip);
    var nextVideos = response.videos ?? const <DbRecording>[];

    if (nextTotalCount > 0 && nextVideos.isEmpty && nextSkip >= nextTotalCount) {
      final lastPageSkip = ((nextTotalCount - 1) ~/ math.max(nextTake, 1)) * nextTake;
      response = await _api.filterVideos(sortColumn: _sortColumn, sortOrder: _sortOrder, take: nextTake, skip: lastPageSkip);
      if (_isStale(generation)) {
        return;
      }
      nextSkip = response.skip ?? lastPageSkip;
      nextTake = response.take ?? nextTake;
      nextTotalCount = response.totalCount ?? nextTotalCount;
      nextVideos = response.videos ?? const <DbRecording>[];
    }

    _skip = nextSkip;
    _pageSize = nextTake;
    _totalCount = math.max(nextTotalCount, nextVideos.length + nextSkip);
    _videos = nextVideos;
    await _savePreferences();
  }

  bool _isStale(int generation) => _disposed || generation != _refreshGeneration;

  void _handleSocketEvent(dynamic event) {
    if (_disposed || event is! SocketEventMessage) {
      return;
    }
    if (event.name == "recording:add") {
      final recording = _socketRecording(event.data);
      if (recording != null &&
          _section == LibrarySection.latest &&
          _sortColumn == RequestsVideoSortColumn.createdAt &&
          _sortOrder == ModelsSortOrder.desc &&
          _skip == 0) {
        _videos = <DbRecording>[
          recording,
          ..._videos.where((item) => item.recordingId != recording.recordingId),
        ];
        if (_videos.length > _pageSize) {
          _videos = _videos.take(_pageSize).toList(growable: false);
        }
        _totalCount += 1;
        _notifySafely();
        return;
      }
    }

    if (event.name.startsWith("recording:") || event.name.startsWith("job:preview:") || event.name == "job:done") {
      _scheduleRefresh();
    }
  }

  void _scheduleRefresh() {
    _refreshDebounce?.cancel();
    _refreshDebounce = Timer(const Duration(milliseconds: 350), refresh);
  }

  DbRecording _videoWithBookmark(
    DbRecording item, {
    required DbRecording target,
    required bool bookmark,
  }) {
    if (item.recordingId != target.recordingId) {
      return item;
    }
    final json = item.toJson();
    json["bookmark"] = bookmark;
    return DbRecording.fromJson(json);
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

  Future<void> _loadPreferences() async {
    final preferences = await SharedPreferences.getInstance();
    _section = LibrarySection.values.firstWhere(
      (section) => section.name == (preferences.getString(_sectionKey) ?? LibrarySection.latest.name),
      orElse: () => LibrarySection.latest,
    );
    _sortColumn = RequestsVideoSortColumn.values.firstWhere(
      (column) => column.name == (preferences.getString(_sortColumnKey) ?? RequestsVideoSortColumn.createdAt.name),
      orElse: () => RequestsVideoSortColumn.createdAt,
    );
    _sortOrder = ModelsSortOrder.values.firstWhere(
      (order) => order.name == (preferences.getString(_sortOrderKey) ?? ModelsSortOrder.desc.name),
      orElse: () => ModelsSortOrder.desc,
    );
    _pageSize = preferences.getInt(_pageSizeKey) ?? defaultPageSize;
    if (!pageSizeOptions.contains(_pageSize)) {
      _pageSize = defaultPageSize;
    }
    _skip = preferences.getInt(_skipKey) ?? 0;
    _randomLimit = preferences.getInt(_randomLimitKey) ?? defaultRandomLimit;
    if (!pageSizeOptions.contains(_randomLimit)) {
      _randomLimit = defaultRandomLimit;
    }
    _bookmarkChannelFilter = preferences.getString(_bookmarkChannelFilterKey) ?? "";
  }

  Future<void> _savePreferences() async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_sectionKey, _section.name);
    await preferences.setString(_sortColumnKey, _sortColumn.name);
    await preferences.setString(_sortOrderKey, _sortOrder.name);
    await preferences.setInt(_pageSizeKey, _pageSize);
    await preferences.setInt(_skipKey, _skip);
    await preferences.setInt(_randomLimitKey, _randomLimit);
    await preferences.setString(_bookmarkChannelFilterKey, _bookmarkChannelFilter);
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
