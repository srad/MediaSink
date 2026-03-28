import "dart:async";

import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/models.dart";
import "package:mediasink_app/app/session_storage.dart";
import "package:mediasink_app/app/session_controller.dart";

import "test_support/app_test_support.dart";

class _ControllableMediaSinkApi extends FakeMediaSinkApi {
  _ControllableMediaSinkApi({required super.config, super.token, super.onUnauthorized, super.channels, super.loginToken, this.failLogin = false, this.failGetChannels = false});

  final bool failLogin;
  final bool failGetChannels;

  @override
  Future<String> login({required String username, required String password}) async {
    if (failLogin) {
      throw Exception("Invalid credentials");
    }
    return super.login(username: username, password: password);
  }

  @override
  Future<List<ServicesChannelInfo>> getChannels() async {
    if (failGetChannels) {
      throw Exception("Request failed");
    }
    return super.getChannels();
  }
}

class _SessionFixture {
  _SessionFixture({Map<String, String> storageValues = const <String, String>{}, this.buildInfo = testBuildInfo, this.channels = const <ServicesChannelInfo>[], this.loginToken = "test-token", this.failStoredTokenValidation = false, this.failLogin = false}) : storage = MemorySessionStorage(storageValues);

  final MemorySessionStorage storage;
  final AppBuildInfo buildInfo;
  final List<ServicesChannelInfo> channels;
  final String loginToken;
  final bool failStoredTokenValidation;
  final bool failLogin;
  final List<_ControllableMediaSinkApi> createdApis = <_ControllableMediaSinkApi>[];

  AppSessionController createController({bool autoBootstrap = false}) {
    return AppSessionController(
      storage: storage,
      autoBootstrap: autoBootstrap,
      buildInfoLoader: (_) async => buildInfo,
      apiFactory: ({required config, token, onUnauthorized}) {
        final api = _ControllableMediaSinkApi(config: config, token: token, onUnauthorized: onUnauthorized, channels: channels, loginToken: loginToken, failLogin: failLogin && token == null, failGetChannels: failStoredTokenValidation && token != null);
        createdApis.add(api);
        return api;
      },
      socketFactory: ({required config, required token}) {
        return FakeMediaSinkSocketService(config: config, token: token, initialState: SocketConnectionState.connected);
      },
    );
  }
}

class _ThrowingSessionStorage implements AppSessionStorage {
  const _ThrowingSessionStorage(this.error);

  final Object error;

  @override
  Future<String?> read({required String key}) async => throw error;

  @override
  Future<void> write({required String key, required String value}) async => throw error;

  @override
  Future<void> delete({required String key}) async => throw error;
}

class _HangingSessionStorage implements AppSessionStorage {
  const _HangingSessionStorage();

  @override
  Future<String?> read({required String key}) => Completer<String?>().future;

  @override
  Future<void> write({required String key, required String value}) => Completer<void>().future;

  @override
  Future<void> delete({required String key}) => Completer<void>().future;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test("bootstrap with stored server and no token reaches logged out with derived config", () async {
    final fixture = _SessionFixture(
      storageValues: storedLoggedOutSession(origin: "http://server-one:3000", username: "alice"),
    );
    final controller = fixture.createController();

    await controller.bootstrap();

    expect(controller.status, SessionStatus.loggedOut);
    expect(controller.savedOrigin, "http://server-one:3000");
    expect(controller.savedUsername, "alice");
    expect(controller.config?.origin, "http://server-one:3000");
    expect(controller.config?.apiBaseUrl, "http://server-one:3000/api/v2");
    expect(controller.token, isNull);
    expect(controller.socket, isNull);
    expect(fixture.createdApis, isEmpty);
  });

  test("bootstrap with stored token authenticates and connects the socket", () async {
    final fixture = _SessionFixture(
      storageValues: storedAuthenticatedSession(origin: "http://server-one:3000", username: "alice", token: "persisted-token"),
      channels: <ServicesChannelInfo>[sampleChannel()],
    );
    final controller = fixture.createController();

    await controller.bootstrap();

    expect(controller.status, SessionStatus.authenticated);
    expect(controller.token, "persisted-token");
    expect(controller.api, isNotNull);
    expect(controller.socket, isNotNull);
    expect(controller.socketConnectionState, SocketConnectionState.connected);
    expect(fixture.createdApis, hasLength(1));
    expect(fixture.createdApis.single.getChannelsCalls, 1);
  });

  test("bootstrap with incompatible api version blocks login and clears stored token", () async {
    const incompatibleBuildInfo = AppBuildInfo(apiVersion: "9.9.9", version: "1.0.0-test", build: "test-build");
    final fixture = _SessionFixture(
      storageValues: storedAuthenticatedSession(token: "persisted-token"),
      buildInfo: incompatibleBuildInfo,
    );
    final controller = fixture.createController();

    await controller.bootstrap();

    expect(controller.status, SessionStatus.incompatibleVersion);
    expect(controller.token, isNull);
    expect(controller.api, isNull);
    expect(controller.socket, isNull);
    expect(controller.message, contains(AppSessionController.supportedApiVersion));
    expect(await fixture.storage.read(key: testTokenKey), isNull);
  });

  test("bootstrap with expired stored token falls back to login and clears token", () async {
    final fixture = _SessionFixture(storageValues: storedAuthenticatedSession(token: "expired-token"), failStoredTokenValidation: true);
    final controller = fixture.createController();

    await controller.bootstrap();

    expect(controller.status, SessionStatus.loggedOut);
    expect(controller.token, isNull);
    expect(controller.api, isNull);
    expect(controller.socket, isNull);
    expect(controller.message, "Stored session expired. Sign in again.");
    expect(await fixture.storage.read(key: testTokenKey), isNull);
  });

  test("configure server normalizes origin, stores it, and clears any token", () async {
    final fixture = _SessionFixture(
      storageValues: storedAuthenticatedSession(origin: "http://old-server:3000", token: "persisted-token"),
    );
    final controller = fixture.createController();

    await controller.configureServer("http://new-server:3000///");

    expect(controller.status, SessionStatus.loggedOut);
    expect(controller.savedOrigin, "http://new-server:3000");
    expect(controller.config?.origin, "http://new-server:3000");
    expect(controller.token, isNull);
    expect(await fixture.storage.read(key: testServerOriginKey), "http://new-server:3000");
    expect(await fixture.storage.read(key: testTokenKey), isNull);
  });

  test("login stores the username and token and transitions to authenticated", () async {
    final fixture = _SessionFixture(
      storageValues: storedLoggedOutSession(origin: "http://server-one:3000", username: "old-user"),
      channels: <ServicesChannelInfo>[sampleChannel()],
      loginToken: "fresh-token",
    );
    final controller = fixture.createController();

    await controller.bootstrap();
    await controller.login(username: "alice", password: "secret");

    expect(controller.status, SessionStatus.authenticated);
    expect(controller.savedUsername, "alice");
    expect(controller.token, "fresh-token");
    expect(controller.socketConnectionState, SocketConnectionState.connected);
    expect(await fixture.storage.read(key: testUsernameKey), "alice");
    expect(await fixture.storage.read(key: testTokenKey), "fresh-token");
  });

  test("login failure returns to logged out without storing a token", () async {
    final fixture = _SessionFixture(storageValues: storedLoggedOutSession(origin: "http://server-one:3000"), failLogin: true);
    final controller = fixture.createController();

    await controller.bootstrap();
    await controller.login(username: "alice", password: "wrong");

    expect(controller.status, SessionStatus.loggedOut);
    expect(controller.message, "Invalid credentials");
    expect(controller.token, isNull);
    expect(controller.socket, isNull);
    expect(await fixture.storage.read(key: testTokenKey), isNull);
  });

  test("unauthorized handler logs out but keeps the configured server", () async {
    final fixture = _SessionFixture(
      storageValues: storedAuthenticatedSession(origin: "http://server-one:3000", username: "alice", token: "persisted-token"),
      channels: <ServicesChannelInfo>[sampleChannel()],
    );
    final controller = fixture.createController();

    await controller.bootstrap();
    await controller.handleUnauthorized();

    expect(controller.status, SessionStatus.loggedOut);
    expect(controller.message, "Your session expired. Sign in again.");
    expect(controller.config?.origin, "http://server-one:3000");
    expect(controller.token, isNull);
    expect(controller.socket, isNull);
    expect(controller.socketConnectionState, SocketConnectionState.disconnected);
    expect(await fixture.storage.read(key: testTokenKey), isNull);
  });

  test("bootstrap migrates legacy storage keys before restoring the session", () async {
    final fixture = _SessionFixture(storageValues: <String, String>{"server_url": "http://legacy-server:3000/", "server_username": "legacy-user", "ws_jwt": "legacy-token"}, channels: <ServicesChannelInfo>[sampleChannel()]);
    final controller = fixture.createController();

    await controller.bootstrap();

    expect(controller.status, SessionStatus.authenticated);
    expect(controller.savedUsername, "legacy-user");
    expect(controller.config?.origin, "http://legacy-server:3000");
    expect(await fixture.storage.read(key: testServerOriginKey), "http://legacy-server:3000");
    expect(await fixture.storage.read(key: testUsernameKey), "legacy-user");
    expect(await fixture.storage.read(key: testTokenKey), "legacy-token");
    expect(await fixture.storage.read(key: "ws_url"), isNull);
  });

  test("bootstrap falls back to unconfigured when storage throws", () async {
    final controller = AppSessionController(storage: _ThrowingSessionStorage(Exception("Storage unavailable")), autoBootstrap: false, buildInfoLoader: (_) async => testBuildInfo);

    await controller.bootstrap();

    expect(controller.status, SessionStatus.unconfigured);
    expect(controller.message, "Storage unavailable");
  });

  test("bootstrap times out hung storage instead of staying on booting", () async {
    final controller = AppSessionController(storage: const _HangingSessionStorage(), storageOperationTimeout: const Duration(milliseconds: 10), autoBootstrap: false, buildInfoLoader: (_) async => testBuildInfo);

    await controller.bootstrap();

    expect(controller.status, SessionStatus.unconfigured);
    expect(controller.message, "Timed out while trying to read app storage.");
  });
}
