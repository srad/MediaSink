import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/api/export.dart";
import "package:mediasink_app/app/app.dart";
import "package:mediasink_app/app/models.dart";
import "package:mediasink_app/app/theme_controller.dart";
import "package:shared_preferences/shared_preferences.dart";

import "test_support/app_test_support.dart";

Future<void> _pumpAuthenticatedApp(WidgetTester tester, {required MediaSinkMobileApp app}) async {
  tester.view.physicalSize = const Size(800, 1000);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  await tester.pumpWidget(app);
  await tester.pumpAndSettle();
  await tester.tap(find.byTooltip("Settings"));
  await tester.pumpAndSettle();
}

Future<void> _scrollToAndTap(WidgetTester tester, Finder finder) async {
  await tester.scrollUntilVisible(
    finder,
    200,
    scrollable: find.byType(Scrollable).first,
  );
  await tester.tap(finder);
  await tester.pumpAndSettle();
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  testWidgets("sign out from settings returns to login", (tester) async {
    final errors = TestErrorTracker()..install();
    try {
      await _pumpAuthenticatedApp(
        tester,
        app: MediaSinkMobileApp(
          sessionController: createTestSessionController(
            storageValues: storedAuthenticatedSession(),
            channels: <ServicesChannelInfo>[sampleChannel()],
            latestVideos: <DbRecording>[sampleVideo()],
            jobs: <DbJob>[sampleJob()],
          ),
          themeController: AppThemeController(autoLoad: false),
        ),
      );

      await _scrollToAndTap(tester, find.widgetWithText(OutlinedButton, "Sign out"));

      expect(find.text("Sign in"), findsWidgets);
      errors.expectClean(tester);
    }
    finally {
      errors.restore();
    }
  });

  testWidgets("reset server from settings returns to server setup", (tester) async {
    final errors = TestErrorTracker()..install();
    try {
      await _pumpAuthenticatedApp(
        tester,
        app: MediaSinkMobileApp(
          sessionController: createTestSessionController(
            storageValues: storedAuthenticatedSession(),
            channels: <ServicesChannelInfo>[sampleChannel()],
            latestVideos: <DbRecording>[sampleVideo()],
            jobs: <DbJob>[sampleJob()],
          ),
          themeController: AppThemeController(autoLoad: false),
        ),
      );

      await _scrollToAndTap(tester, find.text("Server"));

      expect(find.text("Connect to MediaSink"), findsOneWidget);
      errors.expectClean(tester);
    }
    finally {
      errors.restore();
    }
  });

  testWidgets("settings shows the current live connection state", (tester) async {
    final errors = TestErrorTracker()..install();
    final createdSockets = <FakeMediaSinkSocketService>[];
    try {
      await _pumpAuthenticatedApp(
        tester,
        app: MediaSinkMobileApp(
          sessionController: createTestSessionController(
            storageValues: storedAuthenticatedSession(),
            channels: <ServicesChannelInfo>[sampleChannel()],
            latestVideos: <DbRecording>[sampleVideo()],
            jobs: <DbJob>[sampleJob()],
            createdSockets: createdSockets,
          ),
          themeController: AppThemeController(autoLoad: false),
        ),
      );

      createdSockets.single.setConnectionState(SocketConnectionState.reconnecting);
      await tester.pump();

      expect(find.text("Reconnecting"), findsWidgets);
      expect(find.text("Trying to restore live updates."), findsOneWidget);
      errors.expectClean(tester);
    }
    finally {
      errors.restore();
    }
  });
}
