import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../action_confirmation.dart";
import "../models.dart";
import "../session_controller.dart";
import "../theme_controller.dart";
import "../widgets/remote_focusable_action.dart";
import "about_screen.dart";

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key, required this.session});

  final AppSessionController session;

  @override
  Widget build(BuildContext context) {
    final theme = context.watch<AppThemeController>();
    Future<void> closeAnd(Future<void> Function() action) async {
      if (Navigator.of(context).canPop()) {
        Navigator.of(context).pop();
      }
      await action();
    }

    final content = ListView(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
      children: <Widget>[
        _SettingsHeroCard(session: session),
        const SizedBox(height: 12),
        _SettingsSection(
          icon: Icons.palette_outlined,
          title: "Appearance",
          subtitle: "Choose how MediaSink should look on this device.",
          child: SegmentedButton<ThemeMode>(
            segments: const <ButtonSegment<ThemeMode>>[
              ButtonSegment<ThemeMode>(value: ThemeMode.system, icon: Icon(Icons.brightness_auto_rounded), label: Text("System")),
              ButtonSegment<ThemeMode>(value: ThemeMode.light, icon: Icon(Icons.light_mode_rounded), label: Text("Light")),
              ButtonSegment<ThemeMode>(value: ThemeMode.dark, icon: Icon(Icons.dark_mode_rounded), label: Text("Dark")),
            ],
            selected: <ThemeMode>{theme.mode},
            onSelectionChanged: (selection) => theme.setMode(selection.first),
          ),
        ),
        const SizedBox(height: 12),
        _SettingsSection(
          icon: Icons.dns_rounded,
          title: "Connection",
          subtitle: "Current server and session controls.",
          child: Column(
            children: <Widget>[
              _SettingsActionTile(
                icon: _connectionIcon(session.socketConnectionState),
                title: "Server",
                subtitle: session.savedOrigin ?? "Not configured",
                actionLabel: _connectionLabel(session.socketConnectionState),
                onTap: () async {
                  final confirmed = await confirmAction(context, title: "Reset server?", message: "Forget the current server and sign out on this device?", confirmLabel: "Reset", destructive: true);
                  if (!confirmed) {
                    return;
                  }
                  await closeAnd(session.resetServer);
                },
              ),
              const Divider(height: 1),
              _SettingsActionTile(
                icon: Icons.wifi_tethering_rounded,
                title: "Live connection",
                subtitle: _connectionDescription(session.socketConnectionState),
                trailing: _ConnectionPill(state: session.socketConnectionState),
                onTap: () {},
              ),
              const Divider(height: 1),
              _SettingsActionTile(
                icon: Icons.info_outline_rounded,
                title: "About",
                subtitle: "Version and build information",
                trailing: const Icon(Icons.chevron_right_rounded),
                onTap: () {
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const AboutScreen()));
                },
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        _SettingsSection(
          icon: Icons.logout_rounded,
          title: "Session",
          subtitle: "Clear the current session token on this device.",
          child: Row(
            children: <Widget>[
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () async {
                    final confirmed = await confirmAction(context, title: "Sign out?", message: "Clear the current session token on this device?", confirmLabel: "Sign out");
                    if (!confirmed) {
                      return;
                    }
                    await closeAnd(session.logout);
                  },
                  icon: const Icon(Icons.logout_rounded),
                  label: const Text("Sign out"),
                ),
              ),
            ],
          ),
        ),
      ],
    );

    return Scaffold(
      appBar: AppBar(title: const Text("Settings")),
      body: SafeArea(top: false, child: content),
    );
  }
}

IconData _connectionIcon(SocketConnectionState state) {
  return switch (state) {
    SocketConnectionState.connected => Icons.cloud_done_rounded,
    SocketConnectionState.connecting => Icons.cloud_sync_rounded,
    SocketConnectionState.reconnecting => Icons.cloud_sync_rounded,
    SocketConnectionState.disconnected => Icons.cloud_off_rounded,
  };
}

String _connectionLabel(SocketConnectionState state) {
  return switch (state) {
    SocketConnectionState.connected => "Connected",
    SocketConnectionState.connecting => "Connecting",
    SocketConnectionState.reconnecting => "Reconnecting",
    SocketConnectionState.disconnected => "Disconnected",
  };
}

String _connectionDescription(SocketConnectionState state) {
  return switch (state) {
    SocketConnectionState.connected => "Real-time updates are active.",
    SocketConnectionState.connecting => "Opening the live update connection.",
    SocketConnectionState.reconnecting => "Trying to restore live updates.",
    SocketConnectionState.disconnected => "Live updates are currently offline.",
  };
}

class _SettingsHeroCard extends StatelessWidget {
  const _SettingsHeroCard({required this.session});

  final AppSessionController session;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      clipBehavior: Clip.antiAlias,
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          gradient: LinearGradient(colors: <Color>[theme.colorScheme.primary, theme.colorScheme.primary.withValues(alpha: 0.82)], begin: Alignment.topLeft, end: Alignment.bottomRight),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(color: Colors.white.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(12)),
              child: const Icon(Icons.settings_rounded, color: Colors.white),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  const Text(
                    "MediaSink Settings",
                    style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 4),
                  Text(session.savedOrigin ?? "No server configured", style: const TextStyle(color: Colors.white70)),
                  if ((session.savedUsername ?? "").isNotEmpty) ...<Widget>[
                    const SizedBox(height: 8),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      decoration: BoxDecoration(color: Colors.white.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(999)),
                      child: Text(
                        "Signed in as ${session.savedUsername}",
                        style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ConnectionPill extends StatelessWidget {
  const _ConnectionPill({required this.state});

  final SocketConnectionState state;

  @override
  Widget build(BuildContext context) {
    final color = switch (state) {
      SocketConnectionState.connected => Colors.green,
      SocketConnectionState.connecting => Colors.orange,
      SocketConnectionState.reconnecting => Colors.orange,
      SocketConnectionState.disconnected => Theme.of(context).colorScheme.error,
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(color: color.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(999)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(Icons.circle, size: 8, color: color),
          const SizedBox(width: 6),
          Text(_connectionLabel(state), style: Theme.of(context).textTheme.labelMedium?.copyWith(fontWeight: FontWeight.w700)),
        ],
      ),
    );
  }
}

class _SettingsSection extends StatelessWidget {
  const _SettingsSection({required this.icon, required this.title, required this.subtitle, required this.child});

  final IconData icon;
  final String title;
  final String subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                Container(
                  width: 34,
                  height: 34,
                  decoration: BoxDecoration(color: Theme.of(context).colorScheme.primary.withValues(alpha: 0.08), borderRadius: BorderRadius.circular(10)),
                  child: Icon(icon, size: 18, color: Theme.of(context).colorScheme.primary),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      Text(title, style: Theme.of(context).textTheme.titleMedium),
                      const SizedBox(height: 2),
                      Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 14),
            child,
          ],
        ),
      ),
    );
  }
}

class _SettingsActionTile extends StatelessWidget {
  const _SettingsActionTile({required this.icon, required this.title, required this.subtitle, required this.onTap, this.actionLabel, this.trailing});

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final String? actionLabel;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return RemoteFocusableAction(
      onPressed: onTap,
      borderRadius: BorderRadius.circular(10),
      scaleOnFocus: 1.0,
      focusPadding: 0,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 2),
        child: Row(
          children: <Widget>[
            Icon(icon, size: 20),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(title, style: Theme.of(context).textTheme.titleSmall),
                  const SizedBox(height: 2),
                  Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
                ],
              ),
            ),
            if (actionLabel != null)
              Text(
                actionLabel!,
                style: TextStyle(color: Theme.of(context).colorScheme.primary, fontWeight: FontWeight.w600),
              )
            else if (trailing != null)
              trailing!,
          ],
        ),
      ),
    );
  }
}
