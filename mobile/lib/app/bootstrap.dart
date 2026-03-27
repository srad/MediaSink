import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:flutter_native_splash/flutter_native_splash.dart";

import "app.dart";
import "session_controller.dart";
import "theme_controller.dart";

const List<DeviceOrientation> _defaultOrientations = <DeviceOrientation>[
  DeviceOrientation.portraitUp,
  DeviceOrientation.landscapeLeft,
  DeviceOrientation.landscapeRight,
];

Future<void> runMediaSinkApp({
  AppSessionController? sessionController,
  AppThemeController? themeController,
  bool useNativeSplash = true,
  List<DeviceOrientation> preferredOrientations = _defaultOrientations,
}) async {
  final widgetsBinding = WidgetsFlutterBinding.ensureInitialized();
  if (useNativeSplash) {
    FlutterNativeSplash.preserve(widgetsBinding: widgetsBinding);
  }

  await SystemChrome.setPreferredOrientations(preferredOrientations);

  runApp(
    MediaSinkMobileApp(
      sessionController: sessionController,
      themeController: themeController,
    ),
  );

  if (useNativeSplash) {
    FlutterNativeSplash.remove();
  }
}
