import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/app/models.dart";

void main() {
  test("parses build.js metadata", () {
    const script = '''
window.APP_BUILD = "abc123";
window.APP_VERSION = "1.2.3";
window.APP_API_VERSION = "0.1.0";
''';

    final buildInfo = AppBuildInfo.fromBuildJs(script);

    expect(buildInfo.build, "abc123");
    expect(buildInfo.version, "1.2.3");
    expect(buildInfo.apiVersion, "0.1.0");
  });

  test("derives v2 urls from server origin", () {
    const buildInfo = AppBuildInfo(apiVersion: "0.1.0", version: "1.2.3", build: "abc123");

    final config = ServerConfig.create(origin: "https://demo.mediasink.local:3000/", buildInfo: buildInfo);

    expect(config.origin, "https://demo.mediasink.local:3000");
    expect(config.apiBaseUrl, "https://demo.mediasink.local:3000/api/v2");
    expect(config.fileBaseUrl, "https://demo.mediasink.local:3000/videos");
    expect(config.socketUrl, "wss://demo.mediasink.local:3000/api/v2/ws");
  });

  test("preserves primitive socket payloads", () {
    const raw = '{"name":"channel:online","data":42}';

    final message = SocketEventMessage.fromRaw(raw);

    expect(message.name, "channel:online");
    expect(message.data, 42);
  });
}
