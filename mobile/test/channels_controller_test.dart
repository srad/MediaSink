import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/channels_controller.dart";
import "package:mediasink_app/app/models.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  test("channel online and offline socket events patch the stream state locally", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[
        sampleChannel(id: 1, isOnline: false),
      ],
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = ChannelsController(api: api, socket: socket);
    await controller.ready;

    expect(controller.channels.single.isOnline, false);

    socket.emit(const SocketEventMessage(name: "channel:online", data: 1));
    await waitForAsyncTasks();
    expect(controller.channels.single.isOnline, true);

    socket.emit(const SocketEventMessage(name: "channel:offline", data: 1));
    await waitForAsyncTasks();
    expect(controller.channels.single.isOnline, false);
    expect(controller.channels.single.isRecording, false);
  });

  test("thumbnail socket events cache-bust the preview url", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[
        sampleChannel(id: 1, isOnline: true),
      ],
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = ChannelsController(api: api, socket: socket);
    await controller.ready;

    final before = controller.channels.single.preview!;
    socket.emit(const SocketEventMessage(name: "channel:thumbnail", data: 1));
    await waitForAsyncTasks();

    final after = controller.channels.single.preview!;
    expect(after, startsWith(before.split("?").first));
    expect(after, isNot(equals(before)));
    expect(after, contains("?t="));
  });

  test("recording add updates the matching channel stats locally", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[
        sampleChannel(id: 1, isOnline: true),
      ],
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = ChannelsController(api: api, socket: socket);
    await controller.ready;

    socket.emit(
      SocketEventMessage(
        name: "recording:add",
        data: sampleVideo(id: 7).toJson(),
      ),
    );
    await waitForAsyncTasks();

    final channel = controller.channels.single;
    expect(channel.recordingsCount, 2);
    expect(channel.recordingsSize, (sampleVideo(id: 7).size ?? 0));
    expect(channel.recordings?.first.recordingId, 7);
  });

  test("recording add for an unknown channel falls back to a debounced refresh", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[
        sampleChannel(id: 1, isOnline: true),
      ],
    );
    final socket = FakeMediaSinkSocketService(config: api.config, token: "token");
    final controller = ChannelsController(api: api, socket: socket);
    await controller.ready;
    final baselineCalls = api.getChannelsCalls;

    final unknownVideo = DbRecording(
      channelName: "Other Channel",
      filename: "other.mp4",
      pathRelative: "Other Channel/other.mp4",
      videoType: "mp4",
      channelId: 999,
      recordingId: 9,
      size: 100,
    );

    socket.emit(
      SocketEventMessage(
        name: "recording:add",
        data: unknownVideo.toJson(),
      ),
    );
    await waitForDebounce();

    expect(api.getChannelsCalls, baselineCalls + 1);
  });

  test("stream and channel filter preferences survive controller recreation", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[
        sampleChannel(id: 1, name: "Alpha"),
      ],
    );
    final first = ChannelsController(api: api);
    await first.ready;

    first.setStreamSearchQuery("#test alpha");
    first.setStreamFavoritesOnly(true);
    first.setStreamTabIndex(2);
    first.setChannelsSearchQuery("alpha");
    first.setChannelsFavoritesOnly(true);
    first.setChannelsLayout(ChannelListLayout.list);
    first.setChannelsSortField(ChannelsSortField.size);
    first.setChannelsSortDescending(false);
    await waitForAsyncTasks();

    final second = ChannelsController(api: api);
    await second.ready;

    expect(second.streamSearchQuery, "#test alpha");
    expect(second.streamFavoritesOnly, true);
    expect(second.streamTabIndex, 2);
    expect(second.channelsSearchQuery, "alpha");
    expect(second.channelsFavoritesOnly, true);
    expect(second.channelsLayout, ChannelListLayout.list);
    expect(second.channelsSortField, ChannelsSortField.size);
    expect(second.channelsSortDescending, false);
  });

  test("channel list sorting can order by size descending", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[
        const ServicesChannelInfo(channelId: 1, displayName: "Alpha", channelName: "alpha", recordingsCount: 1, recordingsSize: 100),
        const ServicesChannelInfo(channelId: 2, displayName: "Beta", channelName: "beta", recordingsCount: 4, recordingsSize: 900),
        const ServicesChannelInfo(channelId: 3, displayName: "Gamma", channelName: "gamma", recordingsCount: 2, recordingsSize: 400),
      ],
    );
    final controller = ChannelsController(api: api);
    await controller.ready;

    controller.setChannelsSortField(ChannelsSortField.size);
    controller.setChannelsSortDescending(true);

    expect(
      controller.filteredChannels.map((channel) => channel.channelId).toList(growable: false),
      <int?>[2, 3, 1],
    );
  });

  test("toggle recorder pauses and resumes through the api", () async {
    final api = FakeMediaSinkApi(
      config: testServerConfig(),
      channels: <ServicesChannelInfo>[sampleChannel(id: 1)],
      recorderRunning: true,
    );
    final controller = ChannelsController(api: api);
    await controller.ready;

    expect(controller.isRecorderRunning, true);

    await controller.toggleRecorder();
    expect(api.pauseRecorderCalls, 1);
    expect(controller.isRecorderRunning, false);

    await controller.toggleRecorder();
    expect(api.resumeRecorderCalls, 1);
    expect(controller.isRecorderRunning, true);
  });
}
