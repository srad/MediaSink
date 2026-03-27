import "package:flutter_test/flutter_test.dart";
import "package:shared_preferences/shared_preferences.dart";

import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/media_sink_api.dart";
import "package:mediasink_app/app/models.dart";
import "package:mediasink_app/app/playback_progress_controller.dart";

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const buildInfo = AppBuildInfo(
    apiVersion: "0.1.0",
    version: "1.0.0",
    build: "test",
  );

  ServerConfig configFor(String origin) {
    return ServerConfig.create(origin: origin, buildInfo: buildInfo);
  }

  DbRecording recordingFor(int id) {
    return DbRecording(
      channelName: "Channel $id",
      filename: "video_$id.mp4",
      pathRelative: "Channel $id/video_$id.mp4",
      videoType: "mp4",
      recordingId: id,
      duration: 120,
      size: 1024 * id,
    );
  }

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  test("stores and updates playback progress per server", () async {
    final controller = PlaybackProgressController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await controller.ready;

    await controller.recordProgress(
      video: recordingFor(1),
      positionSeconds: 42,
      durationSeconds: 120,
    );

    expect(controller.entries.length, 1);
    expect(controller.entryForVideo(recordingFor(1))?.positionSeconds, 42);
  });

  test("removes completed playback progress", () async {
    final controller = PlaybackProgressController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await controller.ready;

    await controller.recordProgress(
      video: recordingFor(1),
      positionSeconds: 42,
      durationSeconds: 120,
    );
    await controller.recordProgress(
      video: recordingFor(1),
      positionSeconds: 118,
      durationSeconds: 120,
    );

    expect(controller.entries, isEmpty);
  });

  test("keeps playback progress separated by server origin", () async {
    final firstServer = PlaybackProgressController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await firstServer.ready;
    await firstServer.recordProgress(
      video: recordingFor(1),
      positionSeconds: 30,
      durationSeconds: 120,
    );

    final secondServer = PlaybackProgressController(api: MediaSinkApi(config: configFor("http://server-two:3000")));
    await secondServer.ready;

    expect(secondServer.entries, isEmpty);
  });
}
