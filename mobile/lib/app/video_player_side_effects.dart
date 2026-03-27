import "../api/export.dart";
import "history_controller.dart";
import "playback_progress_controller.dart";

class VideoPlayerSideEffects {
  VideoPlayerSideEffects({
    required this.video,
    required this.historyController,
    required this.playbackProgressController,
  });

  final DbRecording video;
  final HistoryController historyController;
  final PlaybackProgressController playbackProgressController;

  bool _historyRecorded = false;
  double _lastPersistedPlaybackSeconds = 0;

  bool get historyRecorded => _historyRecorded;
  double get lastPersistedPlaybackSeconds => _lastPersistedPlaybackSeconds;

  void restorePosition(double seconds) {
    _lastPersistedPlaybackSeconds = seconds.clamp(0, double.infinity);
  }

  Future<void> handlePlaybackTick({
    required bool isPlaying,
    required double positionSeconds,
    required double durationSeconds,
  }) async {
    if (!_historyRecorded && isPlaying && positionSeconds >= HistoryController.playbackThreshold.inSeconds) {
      _historyRecorded = true;
      await historyController.recordPlayedVideo(video);
    }

    if (isPlaying && (positionSeconds - _lastPersistedPlaybackSeconds).abs() >= 15) {
      _lastPersistedPlaybackSeconds = positionSeconds;
      await persistPlaybackProgress(
        positionSeconds: positionSeconds,
        durationSeconds: durationSeconds,
      );
    }
  }

  Future<void> persistPlaybackProgress({
    required double positionSeconds,
    required double durationSeconds,
  }) async {
    _lastPersistedPlaybackSeconds = positionSeconds;
    await playbackProgressController.recordProgress(
      video: video,
      positionSeconds: positionSeconds,
      durationSeconds: durationSeconds,
    );
  }

  Future<void> deleteAndCleanup(Future<void> Function(int recordingId) deleteVideo) async {
    final recordingId = video.recordingId;
    if (recordingId == null) {
      return;
    }

    await deleteVideo(recordingId);
    await historyController.removeByVideo(video);
    await playbackProgressController.removeByVideo(video);
  }
}
