import "package:flutter/foundation.dart";

import "error_utils.dart";
import "media_sink_api.dart";
import "models.dart";
import "socket_service.dart";
import "session_storage.dart";

typedef AppBuildInfoLoader = Future<AppBuildInfo> Function(String origin);
typedef AppSessionApiFactory = MediaSinkApi Function({required ServerConfig config, String? token, UnauthorizedHandler? onUnauthorized});
typedef AppSessionSocketFactory = MediaSinkSocketService Function({required ServerConfig config, required String token});

class AppSessionController extends ChangeNotifier {
  AppSessionController({AppSessionStorage? storage, AppBuildInfoLoader? buildInfoLoader, AppSessionApiFactory? apiFactory, AppSessionSocketFactory? socketFactory, Duration storageOperationTimeout = const Duration(seconds: 5), bool autoBootstrap = true}) : _storage = storage ?? const SecureAppSessionStorage(), _buildInfoLoader = buildInfoLoader ?? MediaSinkApi.fetchBuildInfo, _apiFactory = apiFactory ?? _defaultApiFactory, _storageOperationTimeout = storageOperationTimeout, _socketFactory = socketFactory ?? _defaultSocketFactory {
    if (autoBootstrap) {
      bootstrap();
    }
  }

  static const supportedApiVersion = "0.1.0";

  static const _serverOriginKey = "ms_server_origin";
  static const _usernameKey = "ms_username";
  static const _tokenKey = "ms_token";

  static const _legacyServerUrlKey = "server_url";
  static const _legacyWebSocketUrlKey = "ws_url";
  static const _legacyUsernameKey = "server_username";
  static const _legacyPasswordKey = "server_password";
  static const _legacyJwtKey = "ws_jwt";

  static MediaSinkApi _defaultApiFactory({required ServerConfig config, String? token, UnauthorizedHandler? onUnauthorized}) {
    return MediaSinkApi(config: config, token: token, onUnauthorized: onUnauthorized);
  }

  static MediaSinkSocketService _defaultSocketFactory({required ServerConfig config, required String token}) {
    return MediaSinkSocketService(config: config, token: token);
  }

  final AppSessionStorage _storage;
  final AppBuildInfoLoader _buildInfoLoader;
  final AppSessionApiFactory _apiFactory;
  final AppSessionSocketFactory _socketFactory;
  final Duration _storageOperationTimeout;

  SessionStatus _status = SessionStatus.booting;
  String? _message;
  String? _savedOrigin;
  String? _savedUsername;
  ServerConfig? _config;
  String? _token;
  MediaSinkApi? _api;
  MediaSinkSocketService? _socket;
  SocketConnectionState _socketConnectionState = SocketConnectionState.disconnected;

  SessionStatus get status => _status;
  String? get message => _message;
  String? get savedOrigin => _savedOrigin;
  String? get savedUsername => _savedUsername;
  ServerConfig? get config => _config;
  String? get token => _token;
  MediaSinkApi? get api => _api;
  MediaSinkSocketService? get socket => _socket;
  SocketConnectionState get socketConnectionState => _socketConnectionState;

  Future<void> bootstrap() async {
    _setState(SessionStatus.booting, message: null);
    try {
      await _migrateLegacyStorage();

      _savedOrigin = await _readStorage(_serverOriginKey);
      _savedUsername = await _readStorage(_usernameKey);
      _token = await _readStorage(_tokenKey);

      if (_savedOrigin == null || _savedOrigin!.isEmpty) {
        _setState(SessionStatus.unconfigured, message: null);
        return;
      }

      final buildInfo = await _buildInfoLoader(_savedOrigin!);
      final config = ServerConfig.create(origin: _savedOrigin!, buildInfo: buildInfo);
      _config = config;

      if (config.apiVersion != supportedApiVersion) {
        _token = null;
        await _deleteStorageBestEffort(_tokenKey);
        _setState(SessionStatus.incompatibleVersion, message: "This app expects API $supportedApiVersion but the server reports ${config.apiVersion.isEmpty ? "an empty version" : config.apiVersion}.");
        return;
      }

      if (_token != null && _token!.isNotEmpty) {
        await _activateAuthenticatedSession(token: _token!, username: _savedUsername ?? "", validateToken: true);
        return;
      }

      _setState(SessionStatus.loggedOut, message: null);
    } catch (error) {
      _savedOrigin = null;
      _savedUsername = null;
      _config = null;
      _token = null;
      _api = null;
      await _disposeSocket();
      _setState(SessionStatus.unconfigured, message: friendlyErrorMessage(error));
    }
  }

  Future<void> configureServer(String origin) async {
    _setState(SessionStatus.booting, message: null);
    final normalized = ServerConfig.normalizeOrigin(origin);

    try {
      final buildInfo = await _buildInfoLoader(normalized);
      final config = ServerConfig.create(origin: normalized, buildInfo: buildInfo);
      _savedOrigin = normalized;
      _config = config;
      await _writeStorageBestEffort(_serverOriginKey, normalized);

      if (config.apiVersion != supportedApiVersion) {
        _setState(SessionStatus.incompatibleVersion, message: "The server uses API ${config.apiVersion.isEmpty ? "unknown" : config.apiVersion}; this app supports $supportedApiVersion.");
        return;
      }

      await _deleteStorageBestEffort(_tokenKey);
      _token = null;
      await _disposeSocket();
      _api = null;
      _setState(SessionStatus.loggedOut, message: null);
    } catch (error) {
      _setState(SessionStatus.unconfigured, message: friendlyErrorMessage(error));
    }
  }

  Future<void> login({required String username, required String password}) async {
    final config = _config;
    if (config == null) {
      _setState(SessionStatus.unconfigured, message: "Configure the server first.");
      return;
    }

    _setState(SessionStatus.authenticating, message: null);
    try {
      final api = _apiFactory(config: config);
      final token = await api.login(username: username, password: password);
      await _activateAuthenticatedSession(token: token, username: username.trim(), validateToken: false);
    } catch (error) {
      _setState(SessionStatus.loggedOut, message: friendlyErrorMessage(error));
    }
  }

  Future<void> logout() async {
    await _deleteStorageBestEffort(_tokenKey);
    _token = null;
    _api = null;
    await _disposeSocket();
    _setState(_config == null ? SessionStatus.unconfigured : SessionStatus.loggedOut, message: null);
  }

  Future<void> resetServer() async {
    await logout();
    await _deleteStorageBestEffort(_serverOriginKey);
    _config = null;
    _savedOrigin = null;
    _setState(SessionStatus.unconfigured, message: null);
  }

  Future<void> handleUnauthorized() async {
    await logout();
    _message = "Your session expired. Sign in again.";
    notifyListeners();
  }

  Future<void> _activateAuthenticatedSession({required String token, required String username, required bool validateToken}) async {
    final config = _config;
    if (config == null) {
      _setState(SessionStatus.unconfigured, message: "Configure the server first.");
      return;
    }

    final api = _apiFactory(config: config, token: token, onUnauthorized: handleUnauthorized);

    if (validateToken) {
      try {
        await api.getChannels();
      } catch (_) {
        await _deleteStorageBestEffort(_tokenKey);
        _token = null;
        _api = null;
        _setState(SessionStatus.loggedOut, message: "Stored session expired. Sign in again.");
        return;
      }
    }

    _token = token;
    _savedUsername = username;
    _api = api;

    await _writeStorageBestEffort(_tokenKey, token);
    await _writeStorageBestEffort(_usernameKey, username);

    await _disposeSocket();
    _socket = _socketFactory(config: config, token: token)
      ..addListener(_handleSocketStateChanged)
      ..connect();
    _handleSocketStateChanged();

    _setState(SessionStatus.authenticated, message: null);
  }

  Future<void> _migrateLegacyStorage() async {
    final legacyServerUrl = await _readStorage(_legacyServerUrlKey);
    final legacyUsername = await _readStorage(_legacyUsernameKey);
    final legacyToken = await _readStorage(_legacyJwtKey);

    if ((await _readStorage(_serverOriginKey)) == null && legacyServerUrl != null && legacyServerUrl.isNotEmpty) {
      await _writeStorageBestEffort(_serverOriginKey, ServerConfig.normalizeOrigin(legacyServerUrl));
    }
    if ((await _readStorage(_usernameKey)) == null && legacyUsername != null && legacyUsername.isNotEmpty) {
      await _writeStorageBestEffort(_usernameKey, legacyUsername);
    }
    if ((await _readStorage(_tokenKey)) == null && legacyToken != null && legacyToken.isNotEmpty) {
      await _writeStorageBestEffort(_tokenKey, legacyToken);
    }

    await _deleteStorageBestEffort(_legacyWebSocketUrlKey);
    await _deleteStorageBestEffort(_legacyPasswordKey);
  }

  Future<void> _disposeSocket() async {
    final socket = _socket;
    _socket = null;
    _socketConnectionState = SocketConnectionState.disconnected;
    if (socket != null) {
      socket.removeListener(_handleSocketStateChanged);
      await socket.dispose();
    }
  }

  void _handleSocketStateChanged() {
    final nextState = _socket?.state ?? SocketConnectionState.disconnected;
    if (_socketConnectionState == nextState) {
      return;
    }
    _socketConnectionState = nextState;
    notifyListeners();
  }

  void _setState(SessionStatus status, {required String? message}) {
    _status = status;
    _message = message;
    notifyListeners();
  }

  Future<String?> _readStorage(String key) {
    return _runStorageOperation<String?>("read", key, () => _storage.read(key: key));
  }

  Future<void> _writeStorage(String key, String value) {
    return _runStorageOperation<void>("write", key, () => _storage.write(key: key, value: value));
  }

  Future<void> _deleteStorage(String key) {
    return _runStorageOperation<void>("delete", key, () => _storage.delete(key: key));
  }

  Future<void> _writeStorageBestEffort(String key, String value) async {
    try {
      await _writeStorage(key, value);
    } catch (_) {
      // Keep the current in-memory session usable even when persistence is unavailable.
    }
  }

  Future<void> _deleteStorageBestEffort(String key) async {
    try {
      await _deleteStorage(key);
    } catch (_) {
      // Keep the current in-memory session usable even when persistence is unavailable.
    }
  }

  Future<T> _runStorageOperation<T>(String action, String key, Future<T> Function() operation) async {
    try {
      return await operation().timeout(_storageOperationTimeout, onTimeout: () => throw Exception("Timed out while trying to $action app storage."));
    } catch (error, stackTrace) {
      FlutterError.reportError(FlutterErrorDetails(exception: error, stack: stackTrace, library: "mediasink_app", context: ErrorDescription("while trying to $action session storage for \"$key\"")));
      rethrow;
    }
  }
}
