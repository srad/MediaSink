import "package:flutter/material.dart";
import "package:shared_preferences/shared_preferences.dart";

class AppThemeController extends ChangeNotifier {
  AppThemeController({bool autoLoad = true}) {
    if (autoLoad) {
      _load();
    } else {
      _ready = true;
    }
  }

  static const Color _brandColor = Color(0xFF322448);

  static const _key = "ms_dark_mode";

  ThemeMode _mode = ThemeMode.system;
  bool _ready = false;

  ThemeMode get mode => _mode;
  bool get ready => _ready;

  ThemeData get lightTheme {
    final scheme = ColorScheme.fromSeed(seedColor: _brandColor, brightness: Brightness.light);
    return ThemeData(
      colorScheme: scheme,
      useMaterial3: false,
      scaffoldBackgroundColor: Colors.white,
      primaryColor: _brandColor,
      appBarTheme: const AppBarTheme(backgroundColor: _brandColor, foregroundColor: Colors.white, elevation: 1),
      cardTheme: CardThemeData(
        elevation: 4,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        margin: EdgeInsets.zero,
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(backgroundColor: _brandColor, foregroundColor: Colors.white),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(backgroundColor: _brandColor, foregroundColor: Colors.white),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(foregroundColor: _brandColor),
      ),
      tabBarTheme: const TabBarThemeData(
        labelColor: Colors.white,
        unselectedLabelColor: Colors.white70,
        indicatorColor: Colors.white,
      ),
      bottomNavigationBarTheme: BottomNavigationBarThemeData(
        backgroundColor: Colors.white,
        selectedItemColor: _brandColor,
        unselectedItemColor: scheme.onSurfaceVariant,
        type: BottomNavigationBarType.fixed,
      ),
    );
  }

  ThemeData get darkTheme {
    final scheme = ColorScheme.fromSeed(seedColor: _brandColor, brightness: Brightness.dark);
    return ThemeData(
      colorScheme: scheme,
      useMaterial3: false,
      primaryColor: _brandColor,
      appBarTheme: const AppBarTheme(backgroundColor: _brandColor, foregroundColor: Colors.white, elevation: 1),
      cardTheme: CardThemeData(
        elevation: 4,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        margin: EdgeInsets.zero,
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(backgroundColor: _brandColor, foregroundColor: Colors.white),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(backgroundColor: _brandColor, foregroundColor: Colors.white),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(foregroundColor: Colors.white),
      ),
      tabBarTheme: const TabBarThemeData(
        labelColor: Colors.white,
        unselectedLabelColor: Colors.white70,
        indicatorColor: Colors.white,
      ),
      bottomNavigationBarTheme: BottomNavigationBarThemeData(
        backgroundColor: scheme.surface,
        selectedItemColor: _brandColor,
        unselectedItemColor: scheme.onSurfaceVariant,
        type: BottomNavigationBarType.fixed,
      ),
    );
  }

  Future<void> setMode(ThemeMode mode) async {
    _mode = mode;
    notifyListeners();
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_key, mode.name);
  }

  Future<void> _load() async {
    final preferences = await SharedPreferences.getInstance();
    final raw = preferences.getString(_key);
    _mode = ThemeMode.values.firstWhere((mode) => mode.name == raw, orElse: () => ThemeMode.system);
    _ready = true;
    notifyListeners();
  }
}
