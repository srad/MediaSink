import "package:flutter_test/flutter_test.dart";
import "package:shared_preferences/shared_preferences.dart";

import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/history_controller.dart";
import "package:mediasink_app/app/media_sink_api.dart";
import "package:mediasink_app/app/models.dart";

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

  test("replaying a video moves it to the top without duplicating it", () async {
    final controller = HistoryController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await controller.ready;

    final first = recordingFor(1);
    final second = recordingFor(2);

    await controller.recordPlayedVideo(first);
    await controller.recordPlayedVideo(second);
    await controller.recordPlayedVideo(first);

    expect(controller.entries.length, 2);
    expect(controller.entries.first.video.recordingId, 1);
    expect(controller.entries.last.video.recordingId, 2);
  });

  test("history is trimmed to the newest 100 entries", () async {
    final controller = HistoryController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await controller.ready;

    for (var index = 1; index <= 105; index += 1) {
      await controller.recordPlayedVideo(recordingFor(index));
    }

    expect(controller.entries.length, HistoryController.maxEntries);
    expect(controller.entries.first.video.recordingId, 105);
    expect(controller.entries.last.video.recordingId, 6);
  });

  test("history is stored separately for each server origin", () async {
    final firstServer = HistoryController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await firstServer.ready;
    await firstServer.recordPlayedVideo(recordingFor(1));

    final secondServer = HistoryController(api: MediaSinkApi(config: configFor("http://server-two:3000")));
    await secondServer.ready;

    expect(secondServer.entries, isEmpty);

    await secondServer.recordPlayedVideo(recordingFor(2));

    final firstServerReloaded = HistoryController(api: MediaSinkApi(config: configFor("http://server-one:3000")));
    await firstServerReloaded.ready;
    final secondServerReloaded = HistoryController(api: MediaSinkApi(config: configFor("http://server-two:3000")));
    await secondServerReloaded.ready;

    expect(firstServerReloaded.entries.length, 1);
    expect(firstServerReloaded.entries.single.video.recordingId, 1);
    expect(secondServerReloaded.entries.length, 1);
    expect(secondServerReloaded.entries.single.video.recordingId, 2);
  });
}
