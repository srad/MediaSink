import "package:flutter/material.dart";
import "package:url_launcher/url_launcher.dart";

import "../../api/export.dart";
import "../action_confirmation.dart";
import "../formatters.dart";
import "preview_frame.dart";
import "recording_indicator.dart";
import "remote_focusable_action.dart";

class ClassicStreamCard extends StatelessWidget {
  const ClassicStreamCard({super.key, required this.channel, required this.previewUrl, required this.onOpenDetails, required this.onEdit, required this.onDelete, required this.onTogglePause, required this.onToggleFavorite});

  final ServicesChannelInfo channel;
  final String previewUrl;
  final VoidCallback onOpenDetails;
  final VoidCallback onEdit;
  final VoidCallback onDelete;
  final VoidCallback onTogglePause;
  final VoidCallback onToggleFavorite;

  Future<void> _openExternal() async {
    final raw = channel.url;
    if (raw == null || raw.isEmpty) {
      return;
    }
    final uri = Uri.tryParse(raw);
    if (uri == null) {
      return;
    }
    await launchUrl(uri);
  }

  String get _channelLabel => channel.displayName ?? channel.channelName ?? "channel";

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          RemoteFocusableAction(
            onPressed: onOpenDetails,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(10)),
            child: Stack(
              children: <Widget>[
                ClipRRect(
                  borderRadius: const BorderRadius.vertical(top: Radius.circular(10)),
                  child: PreviewFrame(imageUrl: previewUrl, width: double.infinity, height: 180),
                ),
                if ((channel.isOnline ?? false) == false && (channel.isPaused ?? false) != true)
                  const Positioned.fill(
                    child: Center(child: Icon(Icons.videocam_off_rounded, size: 64, color: Colors.white)),
                  ),
                if (channel.isPaused ?? false)
                  const Positioned.fill(
                    child: Center(child: Icon(Icons.pause_rounded, size: 64, color: Colors.white)),
                  ),
                if (channel.isRecording ?? false) const Positioned(top: 15, right: 15, child: RecordingIndicator()),
                const Positioned(bottom: 15, right: 15, child: Icon(Icons.touch_app_outlined, size: 30, color: Colors.white)),
              ],
            ),
          ),
          Container(
            margin: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                RemoteFocusableAction(
                  onPressed: () {
                    _openExternal();
                  },
                  borderRadius: BorderRadius.circular(8),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 2),
                    child: Row(
                      children: <Widget>[
                        Expanded(
                          child: Text(
                            channel.displayName ?? channel.channelName ?? "Channel",
                            style: TextStyle(fontSize: 20, color: Theme.of(context).colorScheme.primary),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 4),
                        const Icon(Icons.link),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 6),
                SizedBox(
                  height: 36,
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      children: ((channel.tags == null || channel.tags!.isEmpty) ? const <String>["No tags"] : channel.tags!)
                          .map(
                            (tag) => Padding(
                              padding: const EdgeInsets.only(right: 8),
                              child: ElevatedButton(
                                style: ElevatedButton.styleFrom(visualDensity: VisualDensity.compact),
                                onPressed: null,
                                child: Text(tag),
                              ),
                            ),
                          )
                          .toList(),
                    ),
                  ),
                ),
                const SizedBox(height: 4),
                Row(
                  children: <Widget>[
                    Icon(Icons.sd_storage_rounded, color: Theme.of(context).colorScheme.primary, size: 22),
                    const SizedBox(width: 5),
                    Text(formatBytes(channel.recordingsSize)),
                    const SizedBox(width: 10),
                    Icon(Icons.videocam_rounded, color: Theme.of(context).colorScheme.primary, size: 28),
                    const SizedBox(width: 5),
                    Text("${channel.recordingsCount ?? 0}"),
                    const Spacer(),
                    RemoteFocusableAction(
                      onPressed: () async {
                        final confirmed = await confirmAction(context, title: "Delete channel?", message: "Delete $_channelLabel?", confirmLabel: "Delete", destructive: true);
                        if (confirmed) {
                          onDelete();
                        }
                      },
                      borderRadius: BorderRadius.circular(20),
                      focusPadding: 0,
                      scaleOnFocus: 1.08,
                      child: IconButton(
                        visualDensity: VisualDensity.compact,
                        padding: EdgeInsets.zero,
                        onPressed: () async {
                          final confirmed = await confirmAction(context, title: "Delete channel?", message: "Delete $_channelLabel?", confirmLabel: "Delete", destructive: true);
                          if (confirmed) {
                            onDelete();
                          }
                        },
                        icon: const Icon(Icons.delete_rounded),
                      ),
                    ),
                    RemoteFocusableAction(
                      onPressed: onEdit,
                      borderRadius: BorderRadius.circular(20),
                      focusPadding: 0,
                      scaleOnFocus: 1.08,
                      child: IconButton(visualDensity: VisualDensity.compact, padding: EdgeInsets.zero, onPressed: onEdit, icon: const Icon(Icons.edit_rounded)),
                    ),
                    RemoteFocusableAction(
                      onPressed: () async {
                        final confirmed = await confirmAction(context, title: (channel.isPaused ?? false) ? "Resume channel?" : "Pause channel?", message: "${(channel.isPaused ?? false) ? "Resume" : "Pause"} $_channelLabel?", confirmLabel: (channel.isPaused ?? false) ? "Resume" : "Pause");
                        if (confirmed) {
                          onTogglePause();
                        }
                      },
                      borderRadius: BorderRadius.circular(20),
                      focusPadding: 0,
                      scaleOnFocus: 1.08,
                      child: IconButton(
                        visualDensity: VisualDensity.compact,
                        padding: EdgeInsets.zero,
                        onPressed: () async {
                          final confirmed = await confirmAction(context, title: (channel.isPaused ?? false) ? "Resume channel?" : "Pause channel?", message: "${(channel.isPaused ?? false) ? "Resume" : "Pause"} $_channelLabel?", confirmLabel: (channel.isPaused ?? false) ? "Resume" : "Pause");
                          if (confirmed) {
                            onTogglePause();
                          }
                        },
                        icon: Icon((channel.isPaused ?? false) ? Icons.play_circle_fill : Icons.pause_circle_filled_rounded, color: Colors.lightGreen, size: 26),
                      ),
                    ),
                    RemoteFocusableAction(
                      onPressed: onToggleFavorite,
                      borderRadius: BorderRadius.circular(20),
                      focusPadding: 0,
                      scaleOnFocus: 1.08,
                      child: IconButton(
                        visualDensity: VisualDensity.compact,
                        padding: EdgeInsets.zero,
                        onPressed: onToggleFavorite,
                        icon: Icon(Icons.favorite_rounded, color: (channel.fav ?? false) ? Colors.pink : Colors.grey, size: 24),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
