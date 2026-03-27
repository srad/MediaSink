import "package:flutter/material.dart";
import "package:provider/provider.dart";
import "package:provider/single_child_widget.dart";

import "channels_controller.dart";
import "history_controller.dart";
import "jobs_controller.dart";
import "library_controller.dart";
import "media_sink_api.dart";
import "models.dart";
import "playback_progress_controller.dart";
import "screens/login_screen.dart";
import "screens/server_setup_screen.dart";
import "session_controller.dart";
import "socket_service.dart";
import "theme_controller.dart";
import "widgets/app_shell.dart";

class MediaSinkMobileApp extends StatelessWidget {
  const MediaSinkMobileApp({
    super.key,
    this.themeController,
    this.sessionController,
  });

  final AppThemeController? themeController;
  final AppSessionController? sessionController;

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: <SingleChildWidget>[
        if (themeController == null)
          ChangeNotifierProvider<AppThemeController>(create: (_) => AppThemeController())
        else
          ChangeNotifierProvider<AppThemeController>.value(value: themeController!),
        if (sessionController == null)
          ChangeNotifierProvider<AppSessionController>(create: (_) => AppSessionController())
        else
          ChangeNotifierProvider<AppSessionController>.value(value: sessionController!),
      ],
      child: Consumer2<AppThemeController, AppSessionController>(
        builder: (context, theme, session, _) {
          final app = MaterialApp(title: "MediaSink", debugShowCheckedModeBanner: false, theme: theme.lightTheme, darkTheme: theme.darkTheme, themeMode: theme.mode, home: _buildHome(session));
          if (session.status != SessionStatus.authenticated) {
            return app;
          }

          final api = session.api!;
          final socket = session.socket;
          return MultiProvider(
            key: ValueKey<String>("auth-${session.token}"),
            providers: <SingleChildWidget>[
              Provider<MediaSinkApi>.value(value: api),
              if (socket != null) ChangeNotifierProvider<MediaSinkSocketService>.value(value: socket),
              ChangeNotifierProvider<ChannelsController>(
                create: (_) => ChannelsController(api: api, socket: socket),
              ),
              ChangeNotifierProvider<JobsController>(
                create: (_) => JobsController(api: api, socket: socket),
              ),
              ChangeNotifierProvider<LibraryController>(
                create: (_) => LibraryController(api: api, socket: socket),
              ),
              ChangeNotifierProvider<HistoryController>(
                create: (_) => HistoryController(api: api),
              ),
              ChangeNotifierProvider<PlaybackProgressController>(
                create: (_) => PlaybackProgressController(api: api),
              ),
            ],
            child: app,
          );
        },
      ),
    );
  }

  Widget _buildHome(AppSessionController session) {
    switch (session.status) {
      case SessionStatus.booting:
        return const _SplashScreen();
      case SessionStatus.unconfigured:
      case SessionStatus.incompatibleVersion:
        return const ServerSetupScreen();
      case SessionStatus.loggedOut:
      case SessionStatus.authenticating:
        return const LoginScreen();
      case SessionStatus.authenticated:
        return AppShell(session: session);
    }
  }
}

class _SplashScreen extends StatelessWidget {
  const _SplashScreen();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: <Widget>[CircularProgressIndicator(), SizedBox(height: 16), Text("Bootstrapping MediaSink")]),
      ),
    );
  }
}
