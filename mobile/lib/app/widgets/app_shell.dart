import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:provider/provider.dart";

import "../channels_controller.dart";
import "../history_controller.dart";
import "../screens/channels_screen.dart";
import "../screens/channel_editor_sheet.dart";
import "../screens/history_screen.dart";
import "../screens/jobs_screen.dart";
import "../screens/library_screen.dart";
import "../screens/settings_screen.dart";
import "../screens/streams_screen.dart";
import "../session_controller.dart";

class AppShell extends StatefulWidget {
  const AppShell({super.key, required this.session});

  final AppSessionController session;

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  int _index = 0;

  static const _titles = <String>["Streams", "Channels", "Videos", "History", "Jobs"];

  void _switchTab(int delta) {
    final next = (_index + delta).clamp(0, _titles.length - 1);
    if (next != _index) {
      setState(() => _index = next);
    }
  }

  @override
  Widget build(BuildContext context) {
    final pages = <Widget>[const StreamsScreen(), const ChannelsScreen(embedded: true), const LibraryScreen(), const HistoryScreen(), const JobsScreen()];

    return Shortcuts(
      shortcuts: const <ShortcutActivator, Intent>{
        SingleActivator(LogicalKeyboardKey.arrowLeft, alt: true): _SwitchTabIntent(-1),
        SingleActivator(LogicalKeyboardKey.arrowRight, alt: true): _SwitchTabIntent(1),
      },
      child: Actions(
        actions: <Type, Action<Intent>>{
          _SwitchTabIntent: CallbackAction<_SwitchTabIntent>(onInvoke: (intent) {
            _switchTab(intent.delta);
            return null;
          }),
        },
        child: Focus(
          autofocus: true,
          child: Scaffold(
            appBar: AppBar(title: Text(_titles[_index]), actions: _buildAppBarActions(context)),
            body: SafeArea(
              top: false,
              bottom: false,
              child: FocusTraversalGroup(
                policy: ReadingOrderTraversalPolicy(),
                child: IndexedStack(index: _index, children: pages),
              ),
            ),
            bottomNavigationBar: SafeArea(
              top: false,
              child: BottomNavigationBar(
                currentIndex: _index,
                type: BottomNavigationBarType.fixed,
                onTap: (value) => setState(() => _index = value),
                items: const <BottomNavigationBarItem>[
                  BottomNavigationBarItem(icon: Icon(Icons.sensors), label: "Streams"),
                  BottomNavigationBarItem(icon: Icon(Icons.tv), label: "Channels"),
                  BottomNavigationBarItem(icon: Icon(Icons.video_library_outlined), label: "Videos"),
                  BottomNavigationBarItem(icon: Icon(Icons.history_rounded), label: "History"),
                  BottomNavigationBarItem(icon: Icon(Icons.work_outline), label: "Jobs"),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  List<Widget> _buildAppBarActions(BuildContext context) {
    return <Widget>[
      if (_index == 0) ..._buildStreamActions(context),
      if (_index == 3) ..._buildHistoryActions(context),
      IconButton(
        tooltip: "Settings",
        onPressed: () {
          Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => SettingsScreen(session: widget.session)));
        },
        icon: const Icon(Icons.settings_outlined),
      ),
      const SizedBox(width: 4),
    ];
  }

  List<Widget> _buildHistoryActions(BuildContext context) {
    return <Widget>[
      Consumer<HistoryController>(
        builder: (context, controller, _) {
          return IconButton(tooltip: "Clear history", onPressed: controller.loading || controller.entries.isEmpty ? null : () => _confirmClearHistory(context, controller), icon: const Icon(Icons.delete_sweep_rounded));
        },
      ),
    ];
  }

  List<Widget> _buildStreamActions(BuildContext context) {
    return <Widget>[
      Consumer<ChannelsController>(
        builder: (context, controller, _) {
          return Padding(
            padding: const EdgeInsets.only(right: 6),
            child: _AppBarActionPill(
              onPressed: () => _confirmToggleRecorder(context, controller),
              icon: _RecorderActionIcon(isRecording: controller.isRecorderRunning),
              label: Text(controller.isRecorderRunning ? "Pause" : "Resume"),
            ),
          );
        },
      ),
      Padding(
        padding: const EdgeInsets.only(right: 8),
        child: _AppBarActionPill(onPressed: () => openChannelEditorFlow(context), icon: const Icon(Icons.add_rounded, size: 18), label: const Text("Add")),
      ),
    ];
  }

  Future<void> _confirmToggleRecorder(BuildContext context, ChannelsController controller) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text("Confirm"),
          content: Text(controller.isRecorderRunning ? "Pause recording?" : "Resume recording?"),
          actions: <Widget>[
            TextButton(onPressed: () => Navigator.of(dialogContext).pop(false), child: const Text("Cancel")),
            FilledButton(onPressed: () => Navigator.of(dialogContext).pop(true), child: const Text("OK")),
          ],
        );
      },
    );

    if (confirmed != true || !context.mounted) {
      return;
    }

    await controller.toggleRecorder();
  }

  Future<void> _confirmClearHistory(BuildContext context, HistoryController controller) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text("Clear history?"),
          content: const Text("The played video history for this server will be removed from this device."),
          actions: <Widget>[
            TextButton(onPressed: () => Navigator.of(dialogContext).pop(false), child: const Text("Cancel")),
            FilledButton(onPressed: () => Navigator.of(dialogContext).pop(true), child: const Text("Clear")),
          ],
        );
      },
    );

    if (confirmed != true || !context.mounted) {
      return;
    }

    await controller.clearAll();
  }
}

class _AppBarActionPill extends StatelessWidget {
  const _AppBarActionPill({required this.onPressed, required this.icon, required this.label});

  final VoidCallback onPressed;
  final Widget icon;
  final Widget label;

  @override
  Widget build(BuildContext context) {
    final textStyle = Theme.of(context).textTheme.labelMedium?.copyWith(color: Colors.white, fontWeight: FontWeight.w700);

    return Center(
      child: SizedBox(
        height: 36,
        child: TextButton(
          onPressed: onPressed,
          style: TextButton.styleFrom(
            foregroundColor: Colors.white,
            backgroundColor: Colors.black.withValues(alpha: 0.16),
            minimumSize: const Size(0, 36),
            maximumSize: const Size(double.infinity, 36),
            padding: const EdgeInsets.symmetric(horizontal: 11),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            visualDensity: const VisualDensity(horizontal: -1, vertical: -2),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            textStyle: textStyle,
          ),
          child: IconTheme(
            data: const IconThemeData(color: Colors.white, size: 18),
            child: DefaultTextStyle(
              style: textStyle ?? const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 12),
              child: Row(mainAxisSize: MainAxisSize.min, children: <Widget>[icon, const SizedBox(width: 6), label]),
            ),
          ),
        ),
      ),
    );
  }
}

class _RecorderActionIcon extends StatefulWidget {
  const _RecorderActionIcon({required this.isRecording});

  final bool isRecording;

  @override
  State<_RecorderActionIcon> createState() => _RecorderActionIconState();
}

class _RecorderActionIconState extends State<_RecorderActionIcon> with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _opacity;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(vsync: this, duration: const Duration(milliseconds: 900));
    _opacity = Tween<double>(begin: 1, end: 0.25).animate(CurvedAnimation(parent: _controller, curve: Curves.easeInOut));
    _syncAnimation();
  }

  @override
  void didUpdateWidget(covariant _RecorderActionIcon oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.isRecording != widget.isRecording) {
      _syncAnimation();
    }
  }

  void _syncAnimation() {
    if (widget.isRecording) {
      _controller.repeat(reverse: true);
    } else {
      _controller.stop();
      _controller.value = 1;
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.isRecording) {
      return const Icon(Icons.circle, color: Colors.white54, size: 12);
    }

    return FadeTransition(
      opacity: _opacity,
      child: const Icon(Icons.circle, color: Colors.redAccent, size: 12),
    );
  }
}

class _SwitchTabIntent extends Intent {
  const _SwitchTabIntent(this.delta);

  final int delta;
}
