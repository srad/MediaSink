import "dart:async";

import "package:flutter/foundation.dart";
import "package:web_socket_channel/web_socket_channel.dart";

import "models.dart";

class MediaSinkSocketService extends ChangeNotifier {
  MediaSinkSocketService({required this.config, required this.token});

  final ServerConfig config;
  final String token;

  final StreamController<SocketEventMessage> _events = StreamController<SocketEventMessage>.broadcast();

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  Timer? _reconnectTimer;
  bool _manualClose = false;
  int _reconnectAttempts = 0;
  SocketConnectionState _state = SocketConnectionState.disconnected;

  Stream<SocketEventMessage> get events => _events.stream;
  SocketConnectionState get state => _state;

  void connect() {
    _manualClose = false;
    unawaited(_open(isReconnect: false));
  }

  Future<void> disconnect() async {
    _manualClose = true;
    _reconnectTimer?.cancel();
    await _subscription?.cancel();
    await _channel?.sink.close();
    _subscription = null;
    _channel = null;
    _setState(SocketConnectionState.disconnected);
  }

  @override
  Future<void> dispose() async {
    await disconnect();
    await _events.close();
    super.dispose();
  }

  Future<void> _open({required bool isReconnect}) async {
    _setState(isReconnect ? SocketConnectionState.reconnecting : SocketConnectionState.connecting);
    final uri = Uri.parse(config.socketUrl).replace(queryParameters: <String, String>{"Authorization": token, "ApiVersion": config.apiVersion});

    await _subscription?.cancel();
    await _channel?.sink.close();
    _channel = WebSocketChannel.connect(uri);
    _subscription = _channel!.stream.listen(
      (dynamic raw) {
        if (raw is String) {
          try {
            _events.add(SocketEventMessage.fromRaw(raw));
          } catch (_) {
            // Ignore malformed payloads.
          }
        }
      },
      onError: (_) => _scheduleReconnect(),
      onDone: _scheduleReconnect,
      cancelOnError: true,
    );

    try {
      await _channel!.ready;
      _reconnectAttempts = 0;
      _setState(SocketConnectionState.connected);
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (_manualClose) {
      return;
    }

    _reconnectTimer?.cancel();
    _setState(SocketConnectionState.reconnecting);
    final delay = Duration(seconds: _reconnectDelaySeconds());
    _reconnectTimer = Timer(delay, () => unawaited(_open(isReconnect: true)));
  }

  int _reconnectDelaySeconds() {
    const delays = <int>[1, 2, 4, 8, 16];
    final index = _reconnectAttempts.clamp(0, delays.length - 1);
    _reconnectAttempts += 1;
    return delays[index];
  }

  void _setState(SocketConnectionState nextState) {
    if (_state == nextState) {
      return;
    }
    _state = nextState;
    notifyListeners();
  }
}
