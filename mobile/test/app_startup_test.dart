import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/app.dart";
import "package:mediasink_app/app/theme_controller.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

Finder _appBarTitle(String text) {
  return find.descendant(of: find.byType(AppBar), matching: find.text(text));
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  testWidgets("unconfigured startup renders server setup without framework errors", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController();

    await tester.pumpWidget(
      MediaSinkMobileApp(
        sessionController: session,
        themeController: AppThemeController(autoLoad: false),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text("Connect to MediaSink"), findsOneWidget);
    expect(find.text("Verify server"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("stored server without token renders login without framework errors", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController(
      storageValues: storedLoggedOutSession(),
    );

    await tester.pumpWidget(
      MediaSinkMobileApp(
        sessionController: session,
        themeController: AppThemeController(autoLoad: false),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text("Sign in"), findsWidgets);
    expect(find.text("Change server"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("authenticated startup builds the app shell and provider tree cleanly", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController(
      storageValues: storedAuthenticatedSession(),
      channels: <ServicesChannelInfo>[sampleChannel()],
      latestVideos: <DbRecording>[sampleVideo()],
      jobs: <DbJob>[sampleJob()],
      workerProcessing: true,
    );

    await tester.pumpWidget(
      MediaSinkMobileApp(
        sessionController: session,
        themeController: AppThemeController(autoLoad: false),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(BottomNavigationBar), findsOneWidget);
    expect(_appBarTitle("Streams"), findsOneWidget);
    expect(find.byTooltip("Settings"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("settings icon opens the standalone settings screen", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController(
      storageValues: storedAuthenticatedSession(),
      channels: <ServicesChannelInfo>[sampleChannel()],
      latestVideos: <DbRecording>[sampleVideo()],
      jobs: <DbJob>[sampleJob()],
    );

    await tester.pumpWidget(
      MediaSinkMobileApp(
        sessionController: session,
        themeController: AppThemeController(autoLoad: false),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip("Settings"));
    await tester.pumpAndSettle();

    expect(_appBarTitle("Settings"), findsOneWidget);
    errors.expectClean(tester);
  });

  testWidgets("bottom navigation switches between app shell pages", (tester) async {
    final errors = TestErrorTracker()..install();
    addTearDown(errors.restore);

    final session = createTestSessionController(
      storageValues: storedAuthenticatedSession(),
      channels: <ServicesChannelInfo>[sampleChannel()],
      latestVideos: <DbRecording>[sampleVideo()],
      bookmarkedVideos: <DbRecording>[sampleVideo()],
      randomVideos: <DbRecording>[sampleVideo(id: 2)],
      jobs: <DbJob>[sampleJob()],
    );

    await tester.pumpWidget(
      MediaSinkMobileApp(
        sessionController: session,
        themeController: AppThemeController(autoLoad: false),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text("History"));
    await tester.pumpAndSettle();
    expect(_appBarTitle("History"), findsOneWidget);

    await tester.tap(find.text("Jobs"));
    await tester.pumpAndSettle();
    expect(_appBarTitle("Jobs"), findsOneWidget);

    await tester.tap(find.text("Videos"));
    await tester.pumpAndSettle();
    expect(_appBarTitle("Videos"), findsOneWidget);

    await tester.tap(find.text("Channels"));
    await tester.pumpAndSettle();
    expect(_appBarTitle("Channels"), findsOneWidget);

    errors.expectClean(tester);
  });
}
