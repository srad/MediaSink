import "dart:async";
import "dart:math" as math;

import "package:chewie/chewie.dart";
import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:provider/provider.dart";
import "package:video_player/video_player.dart";

import "../../api/export.dart";
import "../history_controller.dart";
import "../media_sink_api.dart";
import "../playback_progress_controller.dart";
import "../video_player_side_effects.dart";

class VideoPlayerPage extends StatefulWidget {
  const VideoPlayerPage({
    super.key,
    required this.title,
    required this.url,
    required this.video,
  });

  final String title;
  final String url;
  final DbRecording video;

  @override
  State<VideoPlayerPage> createState() => _VideoPlayerPageState();
}

enum VideoPlayerResult { deleted }

enum _AnalysisMode { chapters, highlights }

class _AnalysisPoint {
  const _AnalysisPoint({
    required this.label,
    required this.timestamp,
    this.endTime,
    this.intensity,
  });

  final String label;
  final double timestamp;
  final double? endTime;
  final double? intensity;
}

class _VideoPlayerPageState extends State<VideoPlayerPage> with WidgetsBindingObserver {
  VideoPlayerController? _videoController;
  ChewieController? _chewieController;
  final GlobalKey<_TimelineStripeState> _timelineStripeKey = GlobalKey<_TimelineStripeState>();
  bool? _isLandscape;
  late final HistoryController _historyController;
  late final PlaybackProgressController _playbackProgressController;
  late final MediaSinkApi _api;
  late final VideoPlayerSideEffects _sideEffects;

  ResponsesAnalysisResponse? _analysis;
  ResponsesPreviewManifestResponse? _previewManifest;
  String? _analysisError;
  bool _analysisLoading = true;
  _AnalysisMode _analysisMode = _AnalysisMode.highlights;
  int _currentAnalysisIndex = 0;
  int _timelineFollowResetTick = 0;
  bool _timelineControlsRefreshScheduled = false;
  bool _deletingVideo = false;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _historyController = context.read<HistoryController>();
    _playbackProgressController = context.read<PlaybackProgressController>();
    _api = context.read<MediaSinkApi>();
    _sideEffects = VideoPlayerSideEffects(
      video: widget.video,
      historyController: _historyController,
      playbackProgressController: _playbackProgressController,
    );
    _videoController = VideoPlayerController.networkUrl(Uri.parse(widget.url))
      ..addListener(_handlePlaybackStateChanged)
      ..initialize().then((_) async {
        final resumeEntry = _playbackProgressController.entryForVideo(widget.video);
        final durationSeconds = _videoController!.value.duration.inMilliseconds / 1000;
        final resumeSeconds = (resumeEntry?.positionSeconds ?? 0).clamp(0, durationSeconds).toDouble();
        if (resumeSeconds >= PlaybackProgressController.minimumResumePosition.inSeconds &&
            durationSeconds > 0 &&
            (resumeSeconds / durationSeconds) < PlaybackProgressController.completionThreshold) {
          await _videoController!.seekTo(Duration(milliseconds: (resumeSeconds * 1000).round()));
          _sideEffects.restorePosition(resumeSeconds);
        }
        _chewieController = ChewieController(
          videoPlayerController: _videoController!,
          autoPlay: true,
          allowFullScreen: true,
          allowPlaybackSpeedChanging: true,
          deviceOrientationsOnEnterFullScreen: DeviceOrientation.values,
          deviceOrientationsAfterFullScreen: DeviceOrientation.values,
        );
        _syncFullscreenWithOrientation(force: true);
        if (mounted) {
          setState(() {});
        }
      });

    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadAnalysisData();
    });
  }

  @override
  void didChangeMetrics() {
    super.didChangeMetrics();
    _syncFullscreenWithOrientation();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _videoController?.removeListener(_handlePlaybackStateChanged);
    unawaited(_persistPlaybackProgress());
    _chewieController?.dispose();
    _videoController?.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.paused || state == AppLifecycleState.inactive || state == AppLifecycleState.detached) {
      unawaited(_persistPlaybackProgress());
    }
  }

  Future<void> _loadAnalysisData() async {
    final recordingId = widget.video.recordingId;
    if (recordingId == null) {
      if (mounted) {
        setState(() {
          _analysisLoading = false;
        });
      }
      return;
    }

      try {
        final results = await Future.wait<dynamic>(<Future<dynamic>>[
        _api.getAnalysis(recordingId),
        _api.getPreviewManifest(recordingId),
      ]);
      if (!mounted) {
        return;
      }

      setState(() {
        _analysis = results[0] as ResponsesAnalysisResponse?;
        _previewManifest = results[1] as ResponsesPreviewManifestResponse?;
        _analysisError = null;
        _analysisLoading = false;
        _normalizeAnalysisSelection();
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _analysisError = error.toString();
        _analysisLoading = false;
      });
    }
  }

  void _syncFullscreenWithOrientation({bool force = false}) {
    final chewieController = _chewieController;
    if (!mounted || chewieController == null) {
      return;
    }

    final view = View.maybeOf(context) ?? WidgetsBinding.instance.platformDispatcher.views.first;
    final size = view.physicalSize;
    if (size.isEmpty) {
      return;
    }

    final nextIsLandscape = size.width > size.height;
    if (!force && _isLandscape == nextIsLandscape) {
      return;
    }
    _isLandscape = nextIsLandscape;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || _chewieController == null) {
        return;
      }

      if (nextIsLandscape) {
        if (!_chewieController!.isFullScreen) {
          _chewieController!.enterFullScreen();
        }
      } else if (_chewieController!.isFullScreen) {
        _chewieController!.exitFullScreen();
      }
    });
  }

  List<_AnalysisPoint> get _analysisItems {
    final maxDuration = _currentDurationSeconds > 0 ? _currentDurationSeconds : (widget.video.duration?.toDouble() ?? 0);
    final safeMaxDuration = maxDuration > 0 ? maxDuration : double.infinity;

    if (_analysisMode == _AnalysisMode.chapters) {
      final scenes = _analysis?.scenes ?? const <DbSceneInfo>[];
      return scenes.asMap().entries.map((entry) {
        final index = entry.key;
        final scene = entry.value;
        final start = _clampToDuration(scene.startTime?.toDouble() ?? 0, safeMaxDuration);
        final end = _clampToDuration(scene.endTime?.toDouble() ?? start, safeMaxDuration);
        return _AnalysisPoint(
          label: "Chapter ${index + 1} • ${_formatTime(start)}-${_formatTime(end)}",
          timestamp: start,
          endTime: end,
          intensity: scene.changeIntensity?.toDouble(),
        );
      }).toList(growable: false);
    }

    final highlights = _analysis?.highlights ?? const <DbHighlightInfo>[];
    return highlights.asMap().entries.map((entry) {
      final index = entry.key;
      final highlight = entry.value;
      final timestamp = _clampToDuration(highlight.timestamp?.toDouble() ?? highlight.startTime?.toDouble() ?? 0, safeMaxDuration);
      final end = _clampToDuration(highlight.endTime?.toDouble() ?? timestamp, safeMaxDuration);
      final type = (highlight.type == null || highlight.type!.isEmpty) ? "Highlight" : _capitalize(highlight.type!);
      return _AnalysisPoint(
        label: "$type ${index + 1} • ${_formatTime(timestamp)}",
        timestamp: timestamp,
        endTime: end,
        intensity: highlight.intensity?.toDouble(),
      );
    }).toList(growable: false);
  }

  double get _currentDurationSeconds {
    final value = _videoController?.value;
    if (value == null || !value.isInitialized) {
      return 0;
    }
    return value.duration.inMilliseconds / 1000;
  }

  void _normalizeAnalysisSelection() {
    final items = _analysisItems;
    if (items.isEmpty) {
      _currentAnalysisIndex = 0;
      if ((_analysis?.scenes?.isNotEmpty ?? false) && _analysisMode == _AnalysisMode.highlights) {
        _analysisMode = _AnalysisMode.chapters;
      }
      return;
    }
    _currentAnalysisIndex = _currentAnalysisIndex.clamp(0, items.length - 1);
  }

  void _setAnalysisMode(_AnalysisMode mode) {
    if (_analysisMode == mode) {
      return;
    }
    setState(() {
      _analysisMode = mode;
      _currentAnalysisIndex = 0;
      _normalizeAnalysisSelection();
    });
  }

  Future<void> _seekTo(double seconds, {bool resetTimelineFollow = false}) async {
    final controller = _videoController;
    if (controller == null || !controller.value.isInitialized) {
      return;
    }
    if (resetTimelineFollow && mounted) {
      setState(() {
        _timelineFollowResetTick += 1;
      });
    }
    final clamped = _clampToDuration(seconds, _currentDurationSeconds);
    await controller.seekTo(Duration(milliseconds: (clamped * 1000).round()));
  }

  Future<void> _seekBy(double deltaSeconds) async {
    final controller = _videoController;
    if (controller == null || !controller.value.isInitialized) {
      return;
    }
    final current = controller.value.position.inMilliseconds / 1000;
    await _seekTo(current + deltaSeconds, resetTimelineFollow: true);
  }

  Future<void> _goToAnalysisPoint(int index) async {
    final items = _analysisItems;
    if (items.isEmpty) {
      return;
    }
    final safeIndex = index.clamp(0, items.length - 1);
    setState(() {
      _currentAnalysisIndex = safeIndex;
    });
    await _seekTo(items[safeIndex].timestamp, resetTimelineFollow: true);
  }

  String _analysisCounterLabel(List<_AnalysisPoint> items) {
    if (items.isEmpty) {
      return "-";
    }
    final index = _currentAnalysisIndex.clamp(0, items.length - 1);
    return "${index + 1}/${items.length} (${_formatTime(items[index].timestamp)})";
  }

  void _handleTimelineControlsChanged() {
    if (!mounted || _timelineControlsRefreshScheduled) {
      return;
    }
    _timelineControlsRefreshScheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _timelineControlsRefreshScheduled = false;
      if (mounted) {
        setState(() {});
      }
    });
  }

  void _handlePlaybackStateChanged() {
    if (!mounted) {
      return;
    }

    final controller = _videoController;
    if (controller == null) {
      return;
    }

    final value = controller.value;
    if (!value.isInitialized) {
      return;
    }

    final positionSeconds = value.position.inMilliseconds / 1000;
    unawaited(
      _sideEffects.handlePlaybackTick(
        isPlaying: value.isPlaying,
        positionSeconds: positionSeconds,
        durationSeconds: value.duration.inMilliseconds / 1000,
      ),
    );
  }

  Future<void> _persistPlaybackProgress() async {
    final controller = _videoController;
    if (controller == null) {
      return;
    }
    final value = controller.value;
    if (!value.isInitialized) {
      return;
    }

    final positionSeconds = value.position.inMilliseconds / 1000;
    final durationSeconds = value.duration.inMilliseconds / 1000;
    await _sideEffects.persistPlaybackProgress(
      positionSeconds: positionSeconds,
      durationSeconds: durationSeconds,
    );
  }

  Future<void> _confirmDeleteVideo() async {
    final recordingId = widget.video.recordingId;
    if (_deletingVideo || recordingId == null) {
      return;
    }

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text("Delete video?"),
        content: Text("${widget.video.filename} will be deleted."),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: const Text("Delete"),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) {
      return;
    }

    setState(() {
      _deletingVideo = true;
    });

    try {
      await _sideEffects.deleteAndCleanup(_api.deleteVideo);
      if (!mounted) {
        return;
      }
      Navigator.of(context).pop(VideoPlayerResult.deleted);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _deletingVideo = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final canDelete = widget.video.recordingId != null;

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        title: Text(widget.title),
        actions: <Widget>[
          if (canDelete)
            IconButton(
              onPressed: _deletingVideo ? null : _confirmDeleteVideo,
              tooltip: "Delete video",
              icon: _deletingVideo
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.delete_outline_rounded),
            ),
        ],
      ),
      body: SafeArea(
        top: false,
        bottom: false,
        child: Column(
          children: <Widget>[
            Expanded(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: Colors.black,
                  border: Border(
                    bottom: BorderSide(color: theme.colorScheme.outlineVariant.withValues(alpha: 0.35)),
                  ),
                ),
                child: Center(
                  child: _chewieController == null ? const CircularProgressIndicator() : Chewie(controller: _chewieController!),
                ),
              ),
            ),
            _buildAnalysisPanel(context),
          ],
        ),
      ),
    );
  }

  Widget _buildAnalysisPanel(BuildContext context) {
    final mediaQuery = MediaQuery.of(context);
    final theme = Theme.of(context);
    final hasChapters = (_analysis?.scenes?.isNotEmpty ?? false);
    final hasHighlights = (_analysis?.highlights?.isNotEmpty ?? false);
    final hasAnalysisData = hasChapters || hasHighlights;
    final items = _analysisItems;
    final frameUrls = _timelineFrameUrls(_api);
    final hasPreviewTimeline =
        (_previewManifest?.previewPath?.isNotEmpty ?? false) ||
        frameUrls.isNotEmpty;

    return ValueListenableBuilder<VideoPlayerValue>(
      valueListenable: _videoController ?? ValueNotifier<VideoPlayerValue>(VideoPlayerValue(duration: Duration.zero)),
      builder: (context, value, _) {
        final durationSeconds = value.isInitialized ? value.duration.inMilliseconds / 1000 : (widget.video.duration?.toDouble() ?? 0);
        final positionSeconds = value.isInitialized ? value.position.inMilliseconds / 1000 : 0.0;
        final isPlaying = value.isInitialized && value.isPlaying;
        final timelineState = _timelineStripeKey.currentState;
        final isTimelineLocked = timelineState?.isFollowPlayback ?? true;
        final canZoomOut = timelineState?.canZoomOut ?? false;
        final canZoomIn = timelineState?.canZoomIn ?? false;
        final zoomFactor = timelineState?.zoomFactor ?? 1.0;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            if (hasPreviewTimeline)
              _TimelineStripe(
                key: _timelineStripeKey,
                frameUrls: frameUrls,
                previewPath: _previewManifest?.previewPath,
                previewTimestamps: _previewManifest?.timestamps ?? const <int>[],
                fileBaseUrl: context.read<MediaSinkApi>().config.fileBaseUrl,
                durationSeconds: durationSeconds,
                currentPositionSeconds: positionSeconds,
                scenes: _analysis?.scenes ?? const <DbSceneInfo>[],
                highlights: _analysis?.highlights ?? const <DbHighlightInfo>[],
                mode: _analysisMode,
                selectedIndex: _currentAnalysisIndex,
                isPlaying: isPlaying,
                followResetTick: _timelineFollowResetTick,
                onSeek: _seekTo,
                onControlsChanged: _handleTimelineControlsChanged,
              ),
            SafeArea(
              top: false,
              bottom: false,
              child: Padding(
                padding: EdgeInsets.fromLTRB(12, 10, 12, mediaQuery.viewPadding.bottom + 6),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    if (_analysisLoading)
                      const Padding(
                        padding: EdgeInsets.only(top: 6, bottom: 10),
                        child: Center(child: CircularProgressIndicator()),
                      )
                    else if (_analysisError != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 4, bottom: 10),
                        child: Text(_analysisError!, style: TextStyle(color: theme.colorScheme.error)),
                      )
                    else if (hasAnalysisData)
                      Row(
                        children: <Widget>[
                          Expanded(
                            child: SegmentedButton<_AnalysisMode>(
                              style: SegmentedButton.styleFrom(
                                visualDensity: VisualDensity.compact,
                                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                              ),
                              segments: <ButtonSegment<_AnalysisMode>>[
                                if (hasChapters)
                                  const ButtonSegment<_AnalysisMode>(
                                    value: _AnalysisMode.chapters,
                                    label: Text("Chapters"),
                                    icon: Icon(Icons.bookmarks_rounded),
                                  ),
                                if (hasHighlights)
                                  const ButtonSegment<_AnalysisMode>(
                                    value: _AnalysisMode.highlights,
                                    label: Text("Highlights"),
                                    icon: Icon(Icons.auto_awesome_rounded),
                                  ),
                              ],
                              selected: <_AnalysisMode>{_analysisMode},
                              onSelectionChanged: (selection) => _setAnalysisMode(selection.first),
                            ),
                          ),
                          const SizedBox(width: 8),
                          _SmallNavButton(
                            icon: Icons.chevron_left_rounded,
                            enabled: _currentAnalysisIndex > 0,
                            onPressed: () => _goToAnalysisPoint(_currentAnalysisIndex - 1),
                          ),
                          const SizedBox(width: 6),
                          Container(
                            constraints: const BoxConstraints(minWidth: 90),
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.primary.withValues(alpha: 0.08),
                              borderRadius: BorderRadius.circular(10),
                            ),
                            child: Text(
                              _analysisCounterLabel(items),
                              textAlign: TextAlign.center,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: theme.textTheme.labelLarge,
                            ),
                          ),
                          const SizedBox(width: 6),
                          _SmallNavButton(
                            icon: Icons.chevron_right_rounded,
                            enabled: items.isNotEmpty && _currentAnalysisIndex < items.length - 1,
                            onPressed: () => _goToAnalysisPoint(_currentAnalysisIndex + 1),
                          ),
                        ],
                      ),
                    const SizedBox(height: 10),
                    Row(
                      children: <Widget>[
                        if (hasPreviewTimeline) ...<Widget>[
                          _ZoomButton(
                            icon: isTimelineLocked ? Icons.lock_rounded : Icons.lock_open_rounded,
                            enabled: true,
                            onPressed: () => _timelineStripeKey.currentState?.toggleFollowPlayback(),
                          ),
                          const SizedBox(width: 6),
                          _ZoomButton(
                            icon: Icons.remove_rounded,
                            enabled: canZoomOut,
                            onPressed: () => _timelineStripeKey.currentState?.zoomOut(),
                          ),
                          const SizedBox(width: 6),
                          SizedBox(
                            width: 38,
                            child: Text(
                              "${zoomFactor.toStringAsFixed(1)}x",
                              textAlign: TextAlign.center,
                              style: theme.textTheme.labelMedium,
                            ),
                          ),
                          const SizedBox(width: 6),
                          _ZoomButton(
                            icon: Icons.add_rounded,
                            enabled: canZoomIn,
                            onPressed: () => _timelineStripeKey.currentState?.zoomIn(),
                          ),
                        ],
                        const Spacer(),
                        _SkipButton(
                          icon: Icons.replay_30_rounded,
                          label: "30s",
                          onPressed: () => _seekBy(-30),
                        ),
                        const SizedBox(width: 8),
                        _SkipButton(
                          icon: Icons.forward_30_rounded,
                          label: "30s",
                          onPressed: () => _seekBy(30),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ],
        );
      },
    );
  }

  List<String> _timelineFrameUrls(MediaSinkApi api) {
    final manifestPath = _previewManifest?.previewPath;
    final timestamps = _previewManifest?.timestamps ?? const <int>[];
    if (manifestPath != null && manifestPath.isNotEmpty && timestamps.isNotEmpty) {
      final sampleCount = math.min(10, timestamps.length);
      final step = timestamps.length / sampleCount;
      final urls = <String>[];
      for (var i = 0; i < sampleCount; i += 1) {
        final index = math.min((i * step).floor(), timestamps.length - 1);
        urls.add("${api.config.fileBaseUrl}/$manifestPath/${timestamps[index]}.jpg");
      }
      return urls;
    }
    return api.previewFrames(widget.video);
  }
}

class _TimelineStripe extends StatefulWidget {
  const _TimelineStripe({
    super.key,
    required this.frameUrls,
    required this.previewPath,
    required this.previewTimestamps,
    required this.fileBaseUrl,
    required this.durationSeconds,
    required this.currentPositionSeconds,
    required this.scenes,
    required this.highlights,
    required this.mode,
    required this.selectedIndex,
    required this.isPlaying,
    required this.followResetTick,
    required this.onSeek,
    this.onControlsChanged,
  });

  final List<String> frameUrls;
  final String? previewPath;
  final List<int> previewTimestamps;
  final String fileBaseUrl;
  final double durationSeconds;
  final double currentPositionSeconds;
  final List<DbSceneInfo> scenes;
  final List<DbHighlightInfo> highlights;
  final _AnalysisMode mode;
  final int selectedIndex;
  final bool isPlaying;
  final int followResetTick;
  final ValueChanged<double> onSeek;
  final VoidCallback? onControlsChanged;

  @override
  State<_TimelineStripe> createState() => _TimelineStripeState();
}

class _TimelineStripeState extends State<_TimelineStripe> {
  static const double _timelineHeight = 108;
  static const double _targetTileWidth = 96;
  final ScrollController _scrollController = ScrollController();

  double _viewportWidth = 1;
  double _scrollLeft = 0;
  double _pixelsPerSecond = 1;
  bool _zoomInitialized = false;
  bool _followPlayback = true;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_handleScroll);
  }

  @override
  void didUpdateWidget(covariant _TimelineStripe oldWidget) {
    super.didUpdateWidget(oldWidget);
    final sourceChanged = oldWidget.previewPath != widget.previewPath || oldWidget.previewTimestamps.length != widget.previewTimestamps.length;
    final durationChanged = oldWidget.durationSeconds != widget.durationSeconds;
    final shouldResumeFollow = oldWidget.followResetTick != widget.followResetTick || (!oldWidget.isPlaying && widget.isPlaying);
    if (sourceChanged || durationChanged) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        setState(() {
          _zoomInitialized = false;
        });
        _ensureZoomBounds(resetZoom: true);
      });
    } else if (shouldResumeFollow) {
      if (!_followPlayback) {
        setState(() {
          _followPlayback = true;
        });
        widget.onControlsChanged?.call();
      }
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        _maybeCenterCurrentPosition(force: true);
      });
    } else if (_followPlayback && oldWidget.currentPositionSeconds != widget.currentPositionSeconds) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        _maybeCenterCurrentPosition();
      });
    }
  }

  @override
  void dispose() {
    _scrollController
      ..removeListener(_handleScroll)
      ..dispose();
    super.dispose();
  }

  double get _safeDuration => widget.durationSeconds > 0 ? widget.durationSeconds : 1;

  List<double> get _normalizedTimestamps {
    final duration = _safeDuration;
    final values = widget.previewTimestamps
        .map((value) => value.toDouble())
        .where((value) => value.isFinite && value >= 0)
        .map((value) => math.min(value, duration))
        .toSet()
        .toList(growable: false)
      ..sort();
    return values;
  }

  double get _minPixelsPerSecond => math.max(_viewportWidth, 1) / _safeDuration;

  double get _maxPixelsPerSecond {
    final timestamps = _normalizedTimestamps;
    if (timestamps.length < 2) {
      return math.max(_minPixelsPerSecond, _targetTileWidth);
    }

    var minGap = double.infinity;
    for (var i = 1; i < timestamps.length; i += 1) {
      final gap = timestamps[i] - timestamps[i - 1];
      if (gap > 0 && gap < minGap) {
        minGap = gap;
      }
    }

    final safeGap = minGap.isFinite ? math.max(minGap, 0.1) : 1.0;
    return math.max(_minPixelsPerSecond, _targetTileWidth / safeGap);
  }

  double get _timelineWidth => math.max(_safeDuration * _pixelsPerSecond, _viewportWidth);
  bool get isFollowPlayback => _followPlayback;
  bool get canZoomOut => _pixelsPerSecond > _minPixelsPerSecond + 0.01;
  bool get canZoomIn => _pixelsPerSecond < _maxPixelsPerSecond - 0.01;
  double get zoomFactor => _pixelsPerSecond / _minPixelsPerSecond;

  double get _visibleStartSeconds => _clamp(_scrollLeft / _pixelsPerSecond, 0, _safeDuration);

  double get _visibleEndSeconds => _clamp((_scrollLeft + _viewportWidth) / _pixelsPerSecond, 0, _safeDuration);

  double get _playheadViewportX => (widget.currentPositionSeconds * _pixelsPerSecond) - _scrollLeft;

  bool get _isPlayheadVisible => _playheadViewportX >= 0 && _playheadViewportX <= _viewportWidth;

  List<_VisiblePreviewTile> get _visibleTiles {
    final timestamps = _normalizedTimestamps;
    if (widget.previewPath == null || widget.previewPath!.isEmpty || timestamps.isEmpty) {
      return const <_VisiblePreviewTile>[];
    }

    final overscanPx = _viewportWidth * 2;
    final windowStartTime = _clamp((_scrollLeft - overscanPx) / _pixelsPerSecond, 0, _safeDuration);
    final windowEndTime = _clamp((_scrollLeft + _viewportWidth + overscanPx) / _pixelsPerSecond, 0, _safeDuration);
    final bucketDuration = math.max(_targetTileWidth / _pixelsPerSecond, 0.1);
    final visibleStartIndex = math.max(0, _findFirstTimestampAtOrAfter(timestamps, windowStartTime) - 1);
    final visibleEndIndex = math.min(timestamps.length - 1, _findFirstTimestampAtOrAfter(timestamps, windowEndTime + bucketDuration));
    final visibleFrameCount = visibleEndIndex - visibleStartIndex + 1;
    final desiredBucketCount = math.max(1, ((windowEndTime - windowStartTime) / bucketDuration).ceil());

    if (desiredBucketCount >= visibleFrameCount) {
      final tiles = <_VisiblePreviewTile>[];
      for (var index = visibleStartIndex; index <= visibleEndIndex; index += 1) {
        final timestamp = timestamps[index];
        final nextTimestamp = index + 1 < timestamps.length ? timestamps[index + 1] : _safeDuration;
        final tileStartTime = index == 0 ? 0.0 : timestamp;
        final tileEndTime = math.max(nextTimestamp, tileStartTime + 0.1);
        tiles.add(
          _VisiblePreviewTile(
            key: "$timestamp-$index",
            left: tileStartTime * _pixelsPerSecond,
            width: math.max((tileEndTime - tileStartTime) * _pixelsPerSecond, 1),
            src: _previewFrameUrl(timestamp),
          ),
        );
      }
      return tiles;
    }

    final tiles = <_VisiblePreviewTile>[];
    final alignedBucketStart = math.max(0, (windowStartTime / bucketDuration).floor() * bucketDuration);
    for (var bucketStart = alignedBucketStart; bucketStart < windowEndTime; bucketStart += bucketDuration) {
      final bucketEnd = math.min(bucketStart + bucketDuration, _safeDuration);
      final bucketCenter = bucketStart + ((bucketEnd - bucketStart) / 2);
      final chosenIndex = _findNearestTimestampIndex(timestamps, bucketCenter);
      final chosenTimestamp = timestamps[chosenIndex];
      tiles.add(
        _VisiblePreviewTile(
          key: "${bucketStart.toStringAsFixed(3)}-$chosenTimestamp",
          left: bucketStart * _pixelsPerSecond,
          width: math.max((bucketEnd - bucketStart) * _pixelsPerSecond, 1),
            src: _previewFrameUrl(chosenTimestamp),
        ),
      );
    }
    return tiles;
  }

  void _handleScroll() {
    if (!_scrollController.hasClients) {
      return;
    }
    setState(() {
      _scrollLeft = _scrollController.offset;
    });
  }

  bool _handleScrollNotification(ScrollNotification notification) {
    if (notification.metrics.axis != Axis.horizontal) {
      return false;
    }

    final isManualDrag =
        (notification is ScrollStartNotification && notification.dragDetails != null) ||
        (notification is ScrollUpdateNotification && notification.dragDetails != null);
    if (isManualDrag && _followPlayback) {
      setState(() {
        _followPlayback = false;
      });
      widget.onControlsChanged?.call();
    }

    return false;
  }

  void _setFollowPlayback(bool enabled, {bool center = false}) {
    if (_followPlayback == enabled) {
      if (enabled && center) {
        _maybeCenterCurrentPosition(force: true);
      }
      return;
    }

    setState(() {
      _followPlayback = enabled;
    });
    widget.onControlsChanged?.call();

    if (enabled && center) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        _maybeCenterCurrentPosition(force: true);
      });
    }
  }

  void _updateViewport(double width) {
    final nextWidth = math.max(width, 1.0);
    if ((_viewportWidth - nextWidth).abs() < 0.5) {
      return;
    }
    _viewportWidth = nextWidth;
    _ensureZoomBounds(resetZoom: !_zoomInitialized);
  }

  void _ensureZoomBounds({bool resetZoom = false}) {
    final min = _minPixelsPerSecond;
    final max = _maxPixelsPerSecond;
    final defaultZoom = _clamp(_targetTileWidth / 8, min, max);
    final nextPixelsPerSecond = resetZoom || !_zoomInitialized ? defaultZoom : _clamp(_pixelsPerSecond, min, max);

    if (!mounted) {
      _pixelsPerSecond = nextPixelsPerSecond;
      _zoomInitialized = true;
      return;
    }

    setState(() {
      _pixelsPerSecond = nextPixelsPerSecond;
      _zoomInitialized = true;
    });
    widget.onControlsChanged?.call();

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) {
        return;
      }
      final maxScroll = math.max(_timelineWidth - _viewportWidth, 0.0);
      final nextOffset = _scrollController.offset.clamp(0.0, maxScroll).toDouble();
      if ((_scrollController.offset - nextOffset).abs() > 0.5) {
        _scrollController.jumpTo(nextOffset);
      }
      _maybeCenterCurrentPosition();
    });
  }

  void _changeZoom(double factor) {
    final oldPixelsPerSecond = _pixelsPerSecond;
    final nextPixelsPerSecond = _clamp(oldPixelsPerSecond * factor, _minPixelsPerSecond, _maxPixelsPerSecond);
    if ((nextPixelsPerSecond - oldPixelsPerSecond).abs() < 0.01) {
      return;
    }

    final anchorTime = _scrollController.hasClients ? (_scrollController.offset + (_viewportWidth / 2)) / oldPixelsPerSecond : 0.0;
    setState(() {
      _pixelsPerSecond = nextPixelsPerSecond;
      _zoomInitialized = true;
    });
    widget.onControlsChanged?.call();

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) {
        return;
      }
      final targetOffset = _clamp((anchorTime * _pixelsPerSecond) - (_viewportWidth / 2), 0, math.max(_timelineWidth - _viewportWidth, 0));
      _scrollController.jumpTo(targetOffset);
    });
  }

  void _maybeCenterCurrentPosition({bool force = false}) {
    if (!_scrollController.hasClients) {
      return;
    }

    if (!_followPlayback && !force) {
      return;
    }

    final targetOffset = _clamp((widget.currentPositionSeconds * _pixelsPerSecond) - (_viewportWidth / 2), 0, math.max(_timelineWidth - _viewportWidth, 0));
    if ((_scrollController.offset - targetOffset).abs() > (_viewportWidth * 0.35)) {
      _scrollController.animateTo(
        targetOffset,
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOut,
      );
    }
  }

  void _seekFromLocalDx(double localDx) {
    final timelinePosition = _scrollLeft + localDx;
    final seconds = _clamp(timelinePosition / _pixelsPerSecond, 0, _safeDuration);
    _setFollowPlayback(true);
    widget.onSeek(seconds);
  }

  void toggleFollowPlayback() {
    _setFollowPlayback(!_followPlayback, center: true);
  }

  void zoomOut() {
    _changeZoom(0.75);
  }

  void zoomIn() {
    _changeZoom(1.35);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return LayoutBuilder(
      builder: (context, constraints) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) {
            _updateViewport(constraints.maxWidth);
          }
        });

        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            SizedBox(
              height: _timelineHeight,
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTapDown: (details) => _seekFromLocalDx(details.localPosition.dx),
                child: Stack(
                  children: <Widget>[
                    ColoredBox(color: theme.colorScheme.surfaceContainerHighest),
                    NotificationListener<ScrollNotification>(
                      onNotification: _handleScrollNotification,
                      child: SingleChildScrollView(
                        controller: _scrollController,
                        scrollDirection: Axis.horizontal,
                        child: SizedBox(
                          width: _timelineWidth,
                          height: _timelineHeight,
                          child: Stack(
                            fit: StackFit.expand,
                            children: <Widget>[
                              if (_visibleTiles.isEmpty)
                                _TimelinePreviewFallback(frameUrls: widget.frameUrls, theme: theme)
                              else
                                ..._visibleTiles.map(
                                  (tile) => Positioned(
                                    left: tile.left,
                                    top: 0,
                                    width: tile.width + 1,
                                    height: _timelineHeight,
                                    child: Image.network(
                                      tile.src,
                                      fit: BoxFit.cover,
                                      errorBuilder: (context, error, stackTrace) {
                                        return ColoredBox(color: theme.colorScheme.surfaceContainerHighest);
                                      },
                                    ),
                                  ),
                                ),
                              DecoratedBox(
                                decoration: BoxDecoration(
                                  gradient: LinearGradient(
                                    begin: Alignment.topCenter,
                                    end: Alignment.bottomCenter,
                                    colors: <Color>[
                                      Colors.black.withValues(alpha: 0.05),
                                      Colors.black.withValues(alpha: 0.20),
                                    ],
                                  ),
                                ),
                              ),
                              if (widget.mode == _AnalysisMode.chapters)
                                ..._buildSceneOverlays()
                              else
                                ..._buildHighlightOverlays(),
                              Positioned(
                                left: _clamp(widget.currentPositionSeconds * _pixelsPerSecond, 0, _timelineWidth),
                                top: 0,
                                bottom: 0,
                                child: Container(width: 3, color: Colors.white),
                              ),
                              Positioned(
                                left: 0,
                                right: 0,
                                bottom: 0,
                                child: Container(
                                  height: 8,
                                  color: Colors.black.withValues(alpha: 0.28),
                                  child: FractionallySizedBox(
                                    alignment: Alignment.centerLeft,
                                    widthFactor: (_clamp(widget.currentPositionSeconds, 0, _safeDuration) / _safeDuration).clamp(0.0, 1.0),
                                    child: Container(color: theme.colorScheme.primary),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 8),
            SizedBox(
              height: 28,
              child: Stack(
                clipBehavior: Clip.none,
                children: <Widget>[
                  Positioned(
                    left: 0,
                    bottom: 0,
                    child: Text(_formatTime(_visibleStartSeconds), style: theme.textTheme.bodySmall),
                  ),
                  Positioned(
                    right: 0,
                    bottom: 0,
                    child: Text(_formatTime(_visibleEndSeconds), style: theme.textTheme.bodySmall),
                  ),
                  if (_isPlayheadVisible)
                    Positioned(
                      left: _clamp(_playheadViewportX - 32, 0, math.max(_viewportWidth - 64, 0)),
                      bottom: 0,
                      child: SizedBox(
                        width: 64,
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: <Widget>[
                            Container(
                              width: 1,
                              height: 6,
                              color: Colors.white.withValues(alpha: 0.85),
                            ),
                            const SizedBox(height: 2),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                              decoration: BoxDecoration(
                                color: Colors.black.withValues(alpha: 0.55),
                                borderRadius: BorderRadius.circular(999),
                              ),
                              child: Text(
                                _formatTime(widget.currentPositionSeconds),
                                textAlign: TextAlign.center,
                                style: theme.textTheme.labelSmall?.copyWith(color: Colors.white),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ],
        );
      },
    );
  }

  List<Widget> _buildSceneOverlays() {
    final widgets = <Widget>[];
    for (var index = 0; index < widget.scenes.length; index += 1) {
      final scene = widget.scenes[index];
      final start = _clamp(scene.startTime?.toDouble() ?? 0, 0, _safeDuration);
      final end = _clamp(scene.endTime?.toDouble() ?? start, start, _safeDuration);
      final isSelected = index == widget.selectedIndex;
      final boundaryColor = _sceneBoundaryColor(scene.changeIntensity?.toDouble() ?? 0);

      widgets.add(
        _SceneBoundaryMarker(
          left: start * _pixelsPerSecond,
          color: isSelected ? Colors.white : boundaryColor,
          width: isSelected ? 3 : 2,
          opacity: isSelected ? 0.95 : 0.82,
        ),
      );
      if (isSelected) {
        widgets.add(
          _SceneBoundaryMarker(
            left: end * _pixelsPerSecond,
            color: Colors.white,
            width: 3,
            opacity: 0.95,
          ),
        );
      }
    }
    return widgets;
  }

  List<Widget> _buildHighlightOverlays() {
    final widgets = <Widget>[];
    for (var index = 0; index < widget.highlights.length; index += 1) {
      final highlight = widget.highlights[index];
      final start = _clamp(highlight.startTime?.toDouble() ?? highlight.timestamp?.toDouble() ?? 0, 0, _safeDuration);
      final end = _clamp(highlight.endTime?.toDouble() ?? (start + 2), start, _safeDuration);
      final opacity = 0.14 + ((highlight.intensity?.toDouble() ?? 0.5).clamp(0.0, 1.0) * 0.22);
      final isSelected = index == widget.selectedIndex;
      widgets.add(
        Positioned(
          left: start * _pixelsPerSecond,
          width: math.max((end - start) * _pixelsPerSecond, 4),
          top: 0,
          bottom: 8,
          child: Container(
            decoration: BoxDecoration(
              color: (isSelected ? Colors.white : Colors.amberAccent).withValues(alpha: isSelected ? 0.28 : opacity),
            ),
          ),
        ),
      );
    }
    return widgets;
  }

  int _findFirstTimestampAtOrAfter(List<double> timestamps, double target) {
    var low = 0;
    var high = timestamps.length;
    while (low < high) {
      final mid = (low + high) ~/ 2;
      if (timestamps[mid] < target) {
        low = mid + 1;
      } else {
        high = mid;
      }
    }
    return low;
  }

  int _findNearestTimestampIndex(List<double> timestamps, double target) {
    if (timestamps.length <= 1) {
      return 0;
    }
    final nextIndex = _findFirstTimestampAtOrAfter(timestamps, target);
    if (nextIndex <= 0) {
      return 0;
    }
    if (nextIndex >= timestamps.length) {
      return timestamps.length - 1;
    }
    final previousIndex = nextIndex - 1;
    final previousDistance = (timestamps[previousIndex] - target).abs();
    final nextDistance = (timestamps[nextIndex] - target).abs();
    return previousDistance <= nextDistance ? previousIndex : nextIndex;
  }

  String _previewFrameUrl(double timestamp) {
    return "${widget.fileBaseUrl}/${widget.previewPath}/${_formatPreviewTimestamp(timestamp)}.jpg";
  }
}

class _TimelinePreviewFallback extends StatelessWidget {
  const _TimelinePreviewFallback({required this.frameUrls, required this.theme});

  final List<String> frameUrls;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    if (frameUrls.isEmpty) {
      return Center(child: Icon(Icons.image_outlined, color: theme.colorScheme.onSurfaceVariant));
    }

    return Row(
      children: frameUrls
          .map(
            (url) => Expanded(
              child: Image.network(
                url,
                fit: BoxFit.cover,
                height: double.infinity,
                errorBuilder: (context, error, stackTrace) => ColoredBox(color: theme.colorScheme.surfaceContainerHighest),
              ),
            ),
          )
          .toList(growable: false),
    );
  }
}

class _VisiblePreviewTile {
  const _VisiblePreviewTile({
    required this.key,
    required this.left,
    required this.width,
    required this.src,
  });

  final String key;
  final double left;
  final double width;
  final String src;
}

class _SceneBoundaryMarker extends StatelessWidget {
  const _SceneBoundaryMarker({
    required this.left,
    required this.color,
    required this.width,
    required this.opacity,
  });

  final double left;
  final Color color;
  final double width;
  final double opacity;

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: left,
      top: 0,
      bottom: 8,
      child: IgnorePointer(
        child: Opacity(
          opacity: opacity,
          child: Container(
            width: width,
            color: color,
          ),
        ),
      ),
    );
  }
}

class _SmallNavButton extends StatelessWidget {
  const _SmallNavButton({
    required this.icon,
    required this.enabled,
    required this.onPressed,
  });

  final IconData icon;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FilledButton.tonal(
      onPressed: enabled ? onPressed : null,
      style: FilledButton.styleFrom(
        minimumSize: const Size(42, 42),
        padding: EdgeInsets.zero,
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
      child: Icon(icon),
    );
  }
}

class _SkipButton extends StatelessWidget {
  const _SkipButton({
    required this.icon,
    required this.label,
    required this.onPressed,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FilledButton.tonalIcon(
      onPressed: onPressed,
      style: FilledButton.styleFrom(
        minimumSize: const Size(0, 48),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      ),
      icon: Icon(icon, size: 22),
      label: Text(label),
    );
  }
}

class _ZoomButton extends StatelessWidget {
  const _ZoomButton({
    required this.icon,
    required this.enabled,
    required this.onPressed,
  });

  final IconData icon;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Theme.of(context).colorScheme.primary.withValues(alpha: enabled ? 0.10 : 0.04),
      borderRadius: BorderRadius.circular(8),
      child: InkWell(
        onTap: enabled ? onPressed : null,
        borderRadius: BorderRadius.circular(8),
        child: SizedBox(
          width: 34,
          height: 34,
          child: Icon(icon, size: 18, color: enabled ? Theme.of(context).colorScheme.primary : Theme.of(context).disabledColor),
        ),
      ),
    );
  }
}

double _clamp(num value, num min, num max) => math.min(max, math.max(min, value)).toDouble();

double _clampToDuration(double value, double duration) => _clamp(value, 0, duration.isFinite ? duration : value);

Color _sceneBoundaryColor(double intensity) {
  if (intensity < 0.3) {
    return const Color(0xFFFFD54F);
  }
  if (intensity < 0.6) {
    return const Color(0xFFFFA726);
  }
  if (intensity < 0.85) {
    return const Color(0xFFFF7043);
  }
  return const Color(0xFFEF5350);
}

String _formatTime(double seconds) {
  final totalSeconds = seconds.isFinite ? math.max(0, seconds.round()) : 0;
  final hours = totalSeconds ~/ 3600;
  final minutes = (totalSeconds % 3600) ~/ 60;
  final remainingSeconds = totalSeconds % 60;
  if (hours > 0) {
    return "$hours:${minutes.toString().padLeft(2, "0")}:${remainingSeconds.toString().padLeft(2, "0")}";
  }
  return "$minutes:${remainingSeconds.toString().padLeft(2, "0")}";
}

String _capitalize(String value) {
  if (value.isEmpty) {
    return value;
  }
  return value[0].toUpperCase() + value.substring(1);
}

String _formatPreviewTimestamp(double value) {
  if ((value - value.roundToDouble()).abs() < 0.0001) {
    return value.round().toString();
  }

  var text = value.toStringAsFixed(3);
  text = text.replaceFirst(RegExp(r"0+$"), "");
  text = text.replaceFirst(RegExp(r"\.$"), "");
  return text;
}
