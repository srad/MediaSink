import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/app/history_controller.dart";
import "package:mediasink_app/app/playback_progress_controller.dart";
import "package:mediasink_app/app/video_player_side_effects.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  test("records played history only after the playback threshold", () async {
    final api = FakeMediaSinkApi(config: testServerConfig());
    final historyController = HistoryController(api: api);
    final playbackProgressController = PlaybackProgressController(api: api);
    await historyController.ready;
    await playbackProgressController.ready;

    final sideEffects = VideoPlayerSideEffects(
      video: sampleVideo(),
      historyController: historyController,
      playbackProgressController: playbackProgressController,
    );

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 4,
      durationSeconds: 120,
    );
    expect(historyController.entries, isEmpty);

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 5,
      durationSeconds: 120,
    );
    expect(historyController.entries, hasLength(1));

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 30,
      durationSeconds: 120,
    );
    expect(historyController.entries, hasLength(1));
  });

  test("persists playback progress on the 15-second cadence", () async {
    final api = FakeMediaSinkApi(config: testServerConfig());
    final historyController = HistoryController(api: api);
    final playbackProgressController = PlaybackProgressController(api: api);
    await historyController.ready;
    await playbackProgressController.ready;

    final sideEffects = VideoPlayerSideEffects(
      video: sampleVideo(),
      historyController: historyController,
      playbackProgressController: playbackProgressController,
    );

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 10,
      durationSeconds: 120,
    );
    expect(playbackProgressController.entries, isEmpty);

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 16,
      durationSeconds: 120,
    );
    expect(playbackProgressController.entries, hasLength(1));
    expect(playbackProgressController.entries.single.positionSeconds, 16);

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 20,
      durationSeconds: 120,
    );
    expect(playbackProgressController.entries.single.positionSeconds, 16);

    await sideEffects.handlePlaybackTick(
      isPlaying: true,
      positionSeconds: 31,
      durationSeconds: 120,
    );
    expect(playbackProgressController.entries.single.positionSeconds, 31);
  });

  test("completed playback clears stored progress", () async {
    final api = FakeMediaSinkApi(config: testServerConfig());
    final historyController = HistoryController(api: api);
    final playbackProgressController = PlaybackProgressController(api: api);
    await historyController.ready;
    await playbackProgressController.ready;

    final sideEffects = VideoPlayerSideEffects(
      video: sampleVideo(),
      historyController: historyController,
      playbackProgressController: playbackProgressController,
    );

    await sideEffects.persistPlaybackProgress(
      positionSeconds: 80,
      durationSeconds: 120,
    );
    expect(playbackProgressController.entries, hasLength(1));

    await sideEffects.persistPlaybackProgress(
      positionSeconds: 118,
      durationSeconds: 120,
    );
    expect(playbackProgressController.entries, isEmpty);
  });

  test("delete cleanup removes history and playback progress", () async {
    final api = FakeMediaSinkApi(config: testServerConfig());
    final historyController = HistoryController(api: api);
    final playbackProgressController = PlaybackProgressController(api: api);
    final video = sampleVideo();
    await historyController.ready;
    await playbackProgressController.ready;
    await historyController.recordPlayedVideo(video);
    await playbackProgressController.recordProgress(
      video: video,
      positionSeconds: 30,
      durationSeconds: 120,
    );

    final sideEffects = VideoPlayerSideEffects(
      video: video,
      historyController: historyController,
      playbackProgressController: playbackProgressController,
    );

    var deletedId = 0;
    await sideEffects.deleteAndCleanup((recordingId) async {
      deletedId = recordingId;
    });

    expect(deletedId, video.recordingId);
    expect(historyController.entries, isEmpty);
    expect(playbackProgressController.entries, isEmpty);
  });
}
