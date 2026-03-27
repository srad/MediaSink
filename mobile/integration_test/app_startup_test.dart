import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:flutter_test/flutter_test.dart";
import "package:integration_test/integration_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/bootstrap.dart";
import "package:mediasink_app/app/theme_controller.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../test/test_support/app_test_support.dart";

Finder _appBarTitle(String text) {
  return find.descendant(of: find.byType(AppBar), matching: find.text(text));
}

Future<void> _pumpUntilVisible(
  WidgetTester tester,
  Finder finder, {
  Duration step = const Duration(milliseconds: 100),
  int maxPumps = 30,
}) async {
  for (var index = 0; index < maxPumps; index += 1) {
    await tester.pump(step);
    if (finder.evaluate().isNotEmpty) {
      return;
    }
  }
  fail("Timed out waiting for target widget.");
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  testWidgets("app starts to the first usable screen without framework exceptions", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController();

    await runMediaSinkApp(
      sessionController: session,
      themeController: AppThemeController(autoLoad: false),
      useNativeSplash: false,
      preferredOrientations: const <DeviceOrientation>[],
    );
    await _pumpUntilVisible(tester, find.text("Connect to MediaSink"));

    expect(find.text("Connect to MediaSink"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("authenticated startup smoke renders the shell and settings", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController(
      storageValues: storedAuthenticatedSession(),
      channels: <ServicesChannelInfo>[sampleChannel()],
      latestVideos: <DbRecording>[sampleVideo()],
      jobs: <DbJob>[sampleJob()],
      workerProcessing: true,
    );

    await runMediaSinkApp(
      sessionController: session,
      themeController: AppThemeController(autoLoad: false),
      useNativeSplash: false,
      preferredOrientations: const <DeviceOrientation>[],
    );
    await _pumpUntilVisible(tester, _appBarTitle("Streams"));

    expect(_appBarTitle("Streams"), findsOneWidget);

    await tester.tap(find.byTooltip("Settings"));
    await _pumpUntilVisible(tester, _appBarTitle("Settings"));

    expect(_appBarTitle("Settings"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("setup, login, and sign out smoke flow works end to end", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController(
      channels: <ServicesChannelInfo>[sampleChannel()],
      latestVideos: <DbRecording>[sampleVideo()],
      jobs: <DbJob>[sampleJob()],
      workerProcessing: true,
    );

    await runMediaSinkApp(
      sessionController: session,
      themeController: AppThemeController(autoLoad: false),
      useNativeSplash: false,
      preferredOrientations: const <DeviceOrientation>[],
    );
    await _pumpUntilVisible(tester, find.text("Connect to MediaSink"));

    await tester.enterText(find.byType(TextFormField).first, "http://demo.local:3000");
    await tester.tap(find.text("Verify server"));
    await _pumpUntilVisible(tester, find.text("Sign in").first);

    await tester.enterText(find.widgetWithText(TextFormField, "Username"), "alice");
    await tester.enterText(find.widgetWithText(TextFormField, "Password"), "secret");
    await tester.tap(find.text("Sign in").last);
    await _pumpUntilVisible(tester, _appBarTitle("Streams"));

    await tester.tap(find.byTooltip("Settings"));
    await _pumpUntilVisible(tester, _appBarTitle("Settings"));

    await tester.tap(find.text("Sign out"));
    await _pumpUntilVisible(tester, find.text("Sign in").first);

    expect(find.text("Sign in"), findsWidgets);
    errors.expectClean(tester);
  });
}
