import "package:flutter/material.dart";
import "package:package_info_plus/package_info_plus.dart";
import "package:provider/provider.dart";

import "../error_utils.dart";
import "../media_sink_api.dart";
import "../models.dart";
import "../session_controller.dart";
import "../widgets/inline_error_banner.dart";

class AboutScreen extends StatefulWidget {
  const AboutScreen({super.key});

  @override
  State<AboutScreen> createState() => _AboutScreenState();
}

class _AboutScreenState extends State<AboutScreen> {
  PackageInfo? _packageInfo;
  String? _serverVersion;
  String? _serverCommit;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final api = context.read<MediaSinkApi>();
    try {
      final packageInfo = await PackageInfo.fromPlatform();
      final version = await api.getServerVersion();
      if (!mounted) {
        return;
      }
      setState(() {
        _packageInfo = packageInfo;
        _serverVersion = version.version;
        _serverCommit = version.commit;
        _error = null;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _error = friendlyErrorMessage(error);
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<AppSessionController>();
    return Scaffold(
      appBar: AppBar(title: const Text("About")),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.only(top: 12, bottom: 24),
          children: <Widget>[
            if (_error != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                child: InlineErrorBanner(message: _error!, onRetry: _load),
              ),
            Card(
              child: ListTile(title: const Text("App version"), subtitle: Text(_packageInfo == null ? "Loading..." : "${_packageInfo!.version} (${_packageInfo!.buildNumber})")),
            ),
            Card(
              child: ListTile(title: const Text("Server version"), subtitle: Text(_serverVersion == null ? "Loading..." : "$_serverVersion ($_serverCommit)")),
            ),
            Card(
              child: ListTile(title: const Text("API version"), subtitle: Text(session.config?.apiVersion ?? "Unknown")),
            ),
            Card(
              child: ListTile(title: const Text("Live connection"), subtitle: Text(_connectionLabel(session.socketConnectionState))),
            ),
            Card(
              child: ListTile(title: const Text("Server"), subtitle: Text(session.savedOrigin ?? "Unknown")),
            ),
          ],
        ),
      ),
    );
  }
}

String _connectionLabel(SocketConnectionState state) {
  return switch (state) {
    SocketConnectionState.connected => "Connected",
    SocketConnectionState.connecting => "Connecting",
    SocketConnectionState.reconnecting => "Reconnecting",
    SocketConnectionState.disconnected => "Disconnected",
  };
}
