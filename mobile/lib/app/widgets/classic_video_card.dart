import "package:flutter/material.dart";
import "package:timeago/timeago.dart" as timeago;

import "../../api/export.dart";
import "../formatters.dart";
import "interactive_video_preview.dart";

class ClassicVideoCard extends StatelessWidget {
  const ClassicVideoCard({
    super.key,
    required this.video,
    required this.previewUrl,
    required this.previewFrames,
    required this.onOpen,
    required this.onPlay,
    required this.onDownload,
    required this.onDelete,
    required this.onToggleBookmark,
    this.onOpenChannel,
  });

  final DbRecording video;
  final String previewUrl;
  final List<String> previewFrames;
  final VoidCallback onOpen;
  final VoidCallback onPlay;
  final VoidCallback onDownload;
  final VoidCallback onDelete;
  final VoidCallback onToggleBookmark;
  final VoidCallback? onOpenChannel;

  @override
  Widget build(BuildContext context) {
    final createdAt = DateTime.tryParse(video.createdAt ?? "");
    final createdLabel = createdAt == null ? "now" : timeago.format(createdAt, locale: "en_short");
    final theme = Theme.of(context);

    return Card(
      child: Column(
        children: <Widget>[
          InteractiveVideoPreview(
            coverUrl: previewUrl,
            frameUrls: previewFrames,
            onTap: onOpen,
            height: 180,
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            child: Column(
              children: <Widget>[
                Row(mainAxisAlignment: MainAxisAlignment.center, children: <Widget>[const Icon(Icons.timer_rounded, size: 22), const SizedBox(width: 5), Text(formatDuration(video.duration)), const SizedBox(width: 15), const Icon(Icons.sd_storage_rounded, size: 22), const SizedBox(width: 5), Text(formatBytes(video.size)), const SizedBox(width: 15), const Icon(Icons.timelapse_rounded, size: 22), const SizedBox(width: 5), Text(createdLabel)]),
                const SizedBox(height: 4),
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(
                    children: <Widget>[
                      _VideoActionButton(
                        onPressed: onDownload,
                        tooltip: "Download",
                        icon: Icons.download_rounded,
                        foregroundColor: theme.colorScheme.primary,
                      ),
                      const SizedBox(width: 6),
                      _VideoActionButton(
                        onPressed: onDelete,
                        tooltip: "Delete",
                        icon: Icons.delete_rounded,
                        backgroundColor: theme.colorScheme.errorContainer,
                        foregroundColor: theme.colorScheme.onErrorContainer,
                      ),
                      if (onOpenChannel != null) ...<Widget>[
                        const SizedBox(width: 6),
                        _VideoActionButton(
                          onPressed: onOpenChannel!,
                          tooltip: "Open channel",
                          icon: Icons.grid_view_rounded,
                          foregroundColor: theme.colorScheme.primary,
                        ),
                      ],
                      const Spacer(),
                      _VideoActionButton(
                        onPressed: onPlay,
                        tooltip: "Play",
                        icon: Icons.local_movies_rounded,
                        foregroundColor: theme.colorScheme.primary,
                      ),
                      const SizedBox(width: 6),
                      _VideoActionButton(
                        onPressed: onToggleBookmark,
                        tooltip: "Bookmark",
                        icon: (video.bookmark ?? false) ? Icons.favorite_rounded : Icons.favorite_border_rounded,
                        foregroundColor: (video.bookmark ?? false) ? Colors.pink : theme.colorScheme.primary,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _VideoActionButton extends StatelessWidget {
  const _VideoActionButton({
    required this.onPressed,
    required this.tooltip,
    required this.icon,
    this.backgroundColor,
    this.foregroundColor,
  });

  final VoidCallback onPressed;
  final String tooltip;
  final IconData icon;
  final Color? backgroundColor;
  final Color? foregroundColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: backgroundColor ?? theme.colorScheme.primary.withValues(alpha: 0.08),
      borderRadius: BorderRadius.circular(9),
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(9),
        child: Tooltip(
          message: tooltip,
          child: SizedBox(
            width: 34,
            height: 34,
            child: Icon(icon, size: 18, color: foregroundColor ?? theme.colorScheme.primary),
          ),
        ),
      ),
    );
  }
}
