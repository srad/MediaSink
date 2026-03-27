import "dart:async";
import "dart:math" as math;

import "package:flutter/foundation.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../api/export.dart";
import "error_utils.dart";
import "media_sink_api.dart";
import "models.dart";
import "socket_service.dart";

class JobsController extends ChangeNotifier {
  JobsController({required MediaSinkApi api, MediaSinkSocketService? socket}) : _api = api {
    _subscription = socket?.events.listen(_handleSocketEvent);
    _ready = _initialize();
  }

  static const List<int> rowLimitOptions = <int>[10, 25, 50, 100, -1];
  static const int defaultRowLimit = 25;
  static const String _rowLimitKey = "ms_jobs_row_limit";
  static const String _tabKey = "ms_jobs_tab";

  static const Map<JobsTab, DbJobStatus> _historyStateByTab = <JobsTab, DbJobStatus>{
    JobsTab.completed: DbJobStatus.completed,
    JobsTab.error: DbJobStatus.error,
    JobsTab.canceled: DbJobStatus.canceled,
  };

  final MediaSinkApi _api;
  StreamSubscription? _subscription;
  Timer? _refreshDebounce;
  late final Future<void> _ready;
  bool _disposed = false;
  int _refreshGeneration = 0;

  bool _loading = false;
  String? _error;
  bool _workerProcessing = false;
  bool _togglingWorker = false;
  JobsTab _tab = JobsTab.active;
  int _rowLimit = defaultRowLimit;
  final Map<JobsTab, int> _pageByTab = <JobsTab, int>{
    JobsTab.active: 1,
    JobsTab.completed: 1,
    JobsTab.error: 1,
    JobsTab.canceled: 1,
  };
  Set<int> _deletingJobIds = <int>{};
  List<DbJob> _jobs = const <DbJob>[];
  int _totalCount = 0;
  int _runningCount = 0;
  int _queuedCount = 0;

  bool get loading => _loading;
  String? get error => _error;
  bool get workerProcessing => _workerProcessing;
  bool get togglingWorker => _togglingWorker;
  JobsTab get tab => _tab;
  int get rowLimit => _rowLimit;
  int get currentPage => _pageByTab[_tab] ?? 1;
  List<DbJob> get jobs => _jobs;
  int get totalCount => _totalCount;
  int get processingCount => _runningCount;
  int get queuedCount => _queuedCount;
  int get pageSize => _rowLimit == -1 ? math.max(totalCount, 1) : _rowLimit;
  int get totalPages => math.max(1, (totalCount / pageSize).ceil());
  List<DbJob> get pagedJobs => _jobs;
  Future<void> get ready => _ready;

  List<int?> get visiblePageItems {
    final total = totalPages;
    if (total <= 1) {
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

  String get summary {
    if (_tab == JobsTab.active) {
      return "$processingCount running · $queuedCount queued · ${_workerProcessing ? "worker on" : "worker paused"} · ${pagedJobs.length}/$totalCount";
    }
    final label = switch (_tab) {
      JobsTab.completed => "completed",
      JobsTab.error => "errors",
      JobsTab.canceled => "canceled",
      JobsTab.active => "active",
    };
    return "${pagedJobs.length}/$totalCount $label";
  }

  bool isDeletingJob(int? jobId) => jobId != null && _deletingJobIds.contains(jobId);

  Future<void> _initialize() async {
    await _loadPreferences();
    await refresh();
  }

  Future<void> setTab(JobsTab tab) async {
    if (_tab == tab) {
      return;
    }
    _tab = tab;
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> setRowLimit(int value) async {
    if (_rowLimit == value) {
      return;
    }
    _rowLimit = value;
    _pageByTab.updateAll((_, _) => 1);
    _notifySafely();
    await _savePreferences();
    await refresh();
  }

  Future<void> goToPage(int page) async {
    final targetPage = page.clamp(1, totalPages);
    if (currentPage == targetPage) {
      return;
    }
    _pageByTab[_tab] = targetPage;
    _notifySafely();
    await refresh();
  }

  Future<void> nextPage() => goToPage(currentPage + 1);

  Future<void> previousPage() => goToPage(currentPage - 1);

  Future<void> refresh() async {
    if (_disposed) {
      return;
    }

    final generation = ++_refreshGeneration;
    _loading = true;
    _error = null;
    _notifySafely();

    try {
      final workerFuture = _api.getWorkerStatus();
      final responseFuture = _fetchCurrentPage();
      final results = await Future.wait<dynamic>(<Future<dynamic>>[workerFuture, responseFuture], eagerError: true);
      if (_isStale(generation)) {
        return;
      }

      _workerProcessing = results[0] as bool;
      final response = results[1] as ResponsesJobsResponse;
      _jobs = response.jobs ?? const <DbJob>[];
      _totalCount = response.totalCount ?? _jobs.length;
      _runningCount = 0;
      _queuedCount = 0;

      if (_tab == JobsTab.active) {
        final summaryJobs = await _fetchActiveSummaryJobs(totalCount: _totalCount, pageJobs: _jobs);
        if (_isStale(generation)) {
          return;
        }
        _runningCount = summaryJobs.where((job) => job.active ?? false).length;
        _queuedCount = summaryJobs.where((job) => !(job.active ?? false)).length;
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

  Future<void> toggleWorker() async {
    if (_togglingWorker || _disposed) {
      return;
    }
    _togglingWorker = true;
    _notifySafely();

    try {
      if (_workerProcessing) {
        await _api.pauseWorker();
      } else {
        await _api.resumeWorker();
      }
      await refresh();
    } finally {
      if (!_disposed) {
        _togglingWorker = false;
        _notifySafely();
      }
    }
  }

  Future<void> deleteJob(DbJob job) async {
    final jobId = job.jobId;
    if (jobId == null || _deletingJobIds.contains(jobId) || _disposed) {
      return;
    }

    _deletingJobIds = <int>{..._deletingJobIds, jobId};
    _notifySafely();

    try {
      await _api.deleteJob(jobId);
      _jobs = _jobs.where((item) => item.jobId != jobId).toList(growable: false);
      _totalCount = math.max(_totalCount - 1, 0);
      if (_jobs.isEmpty && currentPage > 1) {
        _pageByTab[_tab] = math.max(currentPage - 1, 1);
        await refresh();
        return;
      }
    } finally {
      if (!_disposed) {
        _deletingJobIds = _deletingJobIds.where((id) => id != jobId).toSet();
        _notifySafely();
      }
    }
  }

  Future<ResponsesJobsResponse> _fetchCurrentPage() async {
    final states = _tab == JobsTab.active ? const <DbJobStatus>[DbJobStatus.open] : <DbJobStatus>[_historyStateByTab[_tab]!];
    final sortOrder = DbJobOrder.desc;

    if (_rowLimit == -1) {
      final probe = await _api.getJobs(states: states, sortOrder: sortOrder, take: 1, skip: 0);
      final totalCount = probe.totalCount ?? (probe.jobs?.length ?? 0);
      if (totalCount <= 1) {
        return probe;
      }
      return _api.getJobs(states: states, sortOrder: sortOrder, take: totalCount, skip: 0);
    }

    var page = currentPage;
    while (true) {
      final response = await _api.getJobs(states: states, sortOrder: sortOrder, take: _rowLimit, skip: (page - 1) * _rowLimit);
      final totalCount = response.totalCount ?? 0;
      final totalPages = math.max(1, (totalCount / _rowLimit).ceil());
      final safePage = page.clamp(1, totalPages);
      if (safePage != page) {
        page = safePage;
        _pageByTab[_tab] = safePage;
        continue;
      }
      return response;
    }
  }

  Future<List<DbJob>> _fetchActiveSummaryJobs({
    required int totalCount,
    required List<DbJob> pageJobs,
  }) async {
    if (totalCount <= pageJobs.length) {
      return pageJobs;
    }
    final response = await _api.getJobs(
      states: const <DbJobStatus>[DbJobStatus.open],
      sortOrder: DbJobOrder.desc,
      take: totalCount,
      skip: 0,
    );
    return response.jobs ?? pageJobs;
  }

  void _handleSocketEvent(dynamic event) {
    if (_disposed || event is! SocketEventMessage) {
      return;
    }
    if (event.name.startsWith("job:")) {
      _refreshDebounce?.cancel();
      _refreshDebounce = Timer(const Duration(milliseconds: 350), refresh);
    }
  }

  Future<void> _loadPreferences() async {
    final preferences = await SharedPreferences.getInstance();
    final storedLimit = preferences.getInt(_rowLimitKey);
    if (storedLimit != null && rowLimitOptions.contains(storedLimit)) {
      _rowLimit = storedLimit;
    }
    _tab = JobsTab.values.firstWhere(
      (tab) => tab.name == (preferences.getString(_tabKey) ?? JobsTab.active.name),
      orElse: () => JobsTab.active,
    );
  }

  Future<void> _savePreferences() async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setInt(_rowLimitKey, _rowLimit);
    await preferences.setString(_tabKey, _tab.name);
  }

  bool _isStale(int generation) => _disposed || generation != _refreshGeneration;

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
