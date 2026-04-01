import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/app.dart";
import "package:mediasink_app/app/models.dart";
import "package:mediasink_app/app/session_controller.dart";
import "package:mediasink_app/app/session_storage.dart";
import "package:mediasink_app/app/theme_controller.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

Future<void> _pumpApp(WidgetTester tester, {required MediaSinkMobileApp app}) async {
  await tester.pumpWidget(app);
  await tester.pumpAndSettle();
}

class _WriteDeleteThrowingSessionStorage implements AppSessionStorage {
  _WriteDeleteThrowingSessionStorage([Map<String, String>? initialValues]) : _values = <String, String>{...?initialValues};

  final Map<String, String> _values;

  @override
  Future<String?> read({required String key}) async => _values[key];

  @override
  Future<void> write({required String key, required String value}) async {
    throw Exception("Storage write unavailable");
  }

  @override
  Future<void> delete({required String key}) async {
    throw Exception("Storage delete unavailable");
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  testWidgets("invalid server url shows validation error", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(sessionController: createTestSessionController(), themeController: AppThemeController(autoLoad: false)),
    );

    await tester.enterText(find.byType(TextFormField).first, "not-a-url");
    await tester.tap(find.text("Verify server"));
    await tester.pump();

    expect(find.text("Enter a valid absolute URL."), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("verifying a valid server transitions from setup to login", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(sessionController: createTestSessionController(), themeController: AppThemeController(autoLoad: false)),
    );

    await tester.enterText(find.byType(TextFormField).first, "http://demo.local:3000");
    await tester.tap(find.text("Verify server"));
    await tester.pumpAndSettle();

    expect(find.text("Sign in"), findsWidgets);
    expect(find.text("http://demo.local:3000"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("verifying a valid server still reaches login when persistence is unavailable", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(
        sessionController: createTestSessionController(storage: _WriteDeleteThrowingSessionStorage()),
        themeController: AppThemeController(autoLoad: false),
      ),
    );

    await tester.enterText(find.byType(TextFormField).first, "http://demo.local:3000");
    await tester.tap(find.text("Verify server"));
    await tester.pumpAndSettle();

    expect(find.text("Sign in"), findsWidgets);
    expect(find.text("http://demo.local:3000"), findsOneWidget);
  });

  testWidgets("incompatible api version stays on setup and shows the blocking message", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);
    const incompatibleBuildInfo = AppBuildInfo(apiVersion: "9.9.9", version: "1.0.0-test", build: "test-build");

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(
        sessionController: createTestSessionController(buildInfo: incompatibleBuildInfo),
        themeController: AppThemeController(autoLoad: false),
      ),
    );

    await tester.enterText(find.byType(TextFormField).first, "http://demo.local:3000");
    await tester.tap(find.text("Verify server"));
    await tester.pumpAndSettle();

    expect(find.text("Connect to MediaSink"), findsOneWidget);
    expect(find.textContaining("supports ${AppSessionController.supportedApiVersion}"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("change server from login returns to setup", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(
        sessionController: createTestSessionController(storageValues: storedLoggedOutSession(origin: "http://demo.local:3000")),
        themeController: AppThemeController(autoLoad: false),
      ),
    );

    await tester.tap(find.text("Change server"));
    await tester.pumpAndSettle();
    await tester.tap(find.text("Change").last);
    await tester.pumpAndSettle();

    expect(find.text("Connect to MediaSink"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("failed sign in stays on login and shows the error", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(
        sessionController: createTestSessionController(
          storageValues: storedLoggedOutSession(origin: "http://demo.local:3000"),
          loginError: Exception("Invalid credentials"),
        ),
        themeController: AppThemeController(autoLoad: false),
      ),
    );

    await tester.enterText(find.widgetWithText(TextFormField, "Username"), "alice");
    await tester.enterText(find.widgetWithText(TextFormField, "Password"), "wrong");
    await tester.tap(find.text("Sign in").last);
    await tester.pumpAndSettle();

    expect(find.text("Invalid credentials"), findsOneWidget);
    expect(find.text("Sign in"), findsWidgets);
    errors.expectClean(tester);
  });

  testWidgets("successful sign in reaches the app shell", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(
        sessionController: createTestSessionController(
          storageValues: storedLoggedOutSession(origin: "http://demo.local:3000"),
          channels: <ServicesChannelInfo>[sampleChannel()],
          latestVideos: <DbRecording>[sampleVideo()],
          jobs: <DbJob>[sampleJob()],
        ),
        themeController: AppThemeController(autoLoad: false),
      ),
    );

    await tester.enterText(find.widgetWithText(TextFormField, "Username"), "alice");
    await tester.enterText(find.widgetWithText(TextFormField, "Password"), "secret");
    await tester.tap(find.text("Sign in").last);
    await tester.pumpAndSettle();

    expect(find.byType(BottomNavigationBar), findsOneWidget);
    expect(find.byTooltip("Settings"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("successful sign in still reaches the app shell when persistence is unavailable", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    await _pumpApp(
      tester,
      app: MediaSinkMobileApp(
        sessionController: createTestSessionController(
          storage: _WriteDeleteThrowingSessionStorage(storedLoggedOutSession(origin: "http://demo.local:3000")),
          channels: <ServicesChannelInfo>[sampleChannel()],
          latestVideos: <DbRecording>[sampleVideo()],
          jobs: <DbJob>[sampleJob()],
        ),
        themeController: AppThemeController(autoLoad: false),
      ),
    );

    await tester.enterText(find.widgetWithText(TextFormField, "Username"), "alice");
    await tester.enterText(find.widgetWithText(TextFormField, "Password"), "secret");
    await tester.tap(find.text("Sign in").last);
    await tester.pumpAndSettle();

    expect(find.byType(BottomNavigationBar), findsOneWidget);
    expect(find.byTooltip("Settings"), findsOneWidget);
  });
}
