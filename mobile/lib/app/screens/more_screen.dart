import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../action_confirmation.dart";
import "../channels_controller.dart";
import "../session_controller.dart";
import "about_screen.dart";
import "channels_screen.dart";
import "settings_screen.dart";

class MoreScreen extends StatelessWidget {
  const MoreScreen({super.key, required this.session});

  final AppSessionController session;

  @override
  Widget build(BuildContext context) {
    final channels = context.watch<ChannelsController>();

    return ListView(
      padding: const EdgeInsets.only(top: 12, bottom: 24),
      children: <Widget>[
        Card(
          child: ListTile(
            title: const Text("Channels"),
            subtitle: Text("${channels.channels.length} configured"),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const ChannelsScreen()));
            },
          ),
        ),
        Card(
          child: ListTile(
            title: const Text("Settings"),
            subtitle: const Text("Theme and server setup"),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => SettingsScreen(session: session)));
            },
          ),
        ),
        Card(
          child: ListTile(
            title: const Text("About"),
            subtitle: const Text("Version and build information"),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const AboutScreen()));
            },
          ),
        ),
        Card(
          child: ListTile(
            title: const Text("Sign out"),
            subtitle: const Text("Clear the current session token"),
            trailing: const Icon(Icons.logout),
            onTap: () async {
              final confirmed = await confirmAction(context, title: "Sign out?", message: "Clear the current session token on this device?", confirmLabel: "Sign out");
              if (!confirmed) {
                return;
              }
              await session.logout();
            },
          ),
        ),
      ],
    );
  }
}
