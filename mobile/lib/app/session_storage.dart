import "package:flutter/services.dart";
import "package:flutter_secure_storage/flutter_secure_storage.dart";
import "package:shared_preferences/shared_preferences.dart";

abstract class AppSessionStorage {
  Future<String?> read({required String key});

  Future<void> write({required String key, required String value});

  Future<void> delete({required String key});
}

class SecureAppSessionStorage implements AppSessionStorage {
  const SecureAppSessionStorage([this._storage = const FlutterSecureStorage()]);

  static const _fallbackPrefix = "secure_fallback.";

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read({required String key}) async {
    try {
      final secureValue = await _storage.read(key: key);
      if (secureValue != null) {
        return secureValue;
      }
    } on MissingPluginException {
      // Fall back when the native secure-storage plugin is unavailable.
    } on PlatformException {
      // Some Android release builds fail to initialize the plugin.
    }

    final preferences = await SharedPreferences.getInstance();
    return preferences.getString(_fallbackKey(key));
  }

  @override
  Future<void> write({required String key, required String value}) async {
    final preferences = await SharedPreferences.getInstance();

    try {
      await _storage.write(key: key, value: value);
      await preferences.remove(_fallbackKey(key));
      return;
    } on MissingPluginException {
      // Fall back when the native secure-storage plugin is unavailable.
    } on PlatformException {
      // Some Android release builds fail to initialize the plugin.
    }

    await preferences.setString(_fallbackKey(key), value);
  }

  @override
  Future<void> delete({required String key}) async {
    final preferences = await SharedPreferences.getInstance();

    try {
      await _storage.delete(key: key);
    } on MissingPluginException {
      // Fall back when the native secure-storage plugin is unavailable.
    } on PlatformException {
      // Some Android release builds fail to initialize the plugin.
    }

    await preferences.remove(_fallbackKey(key));
  }

  String _fallbackKey(String key) => "$_fallbackPrefix$key";
}
