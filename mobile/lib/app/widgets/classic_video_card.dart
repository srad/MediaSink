import "package:flutter/material.dart";
import "package:timeago/timeago.dart" as timeago;

import "../../api/export.dart";
import "../action_confirmation.dart";
import "../formatters.dart";
import "interactive_video_preview.dart";
import "remote_focusable_action.dart";

class ClassicVideoCard extends StatelessWidget {
  const ClassicVideoCard({super.key, required this.video, required this.previewUrl, required this.previewFrames, required this.onOpen, required this.onPlay, required this.onDownload, required this.onDelete, required this.onToggleBookmark, this.onOpenChannel});

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
    final videoLabel = video.filename;

    return LayoutBuilder(
      builder: (context, constraints) {
        final compact = constraints.maxWidth < 235;
        final actionButtonSize = compact ? 30.0 : 34.0;
        final actionIconSize = compact ? 16.0 : 18.0;
        final metadataIconSize = compact ? 18.0 : 22.0;
        final metadataStyle = compact ? theme.textTheme.labelMedium : theme.textTheme.bodyMedium;

        return Card(
          child: Column(
            children: <Widget>[
              InteractiveVideoPreview(coverUrl: previewUrl, frameUrls: previewFrames, onTap: onOpen, height: compact ? 164 : 180),
              Padding(
                padding: EdgeInsets.symmetric(horizontal: compact ? 8 : 10, vertical: 4),
                child: Column(
                  children: <Widget>[
                    Wrap(
                      alignment: WrapAlignment.center,
                      spacing: compact ? 10 : 15,
                      runSpacing: 6,
                      children: <Widget>[
                        _VideoMetadataItem(icon: Icons.timer_rounded, iconSize: metadataIconSize, label: formatDuration(video.duration), style: metadataStyle),
                        _VideoMetadataItem(icon: Icons.sd_storage_rounded, iconSize: metadataIconSize, label: formatBytes(video.size), style: metadataStyle),
                        _VideoMetadataItem(icon: Icons.timelapse_rounded, iconSize: metadataIconSize, label: createdLabel, style: metadataStyle),
                      ],
                    ),
                    const SizedBox(height: 6),
                    Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Wrap(
                        alignment: WrapAlignment.center,
                        spacing: compact ? 4 : 6,
                        runSpacing: compact ? 4 : 6,
                        children: <Widget>[
                          _VideoActionButton(onPressed: onDownload, tooltip: "Download", icon: Icons.download_rounded, foregroundColor: theme.colorScheme.primary, size: actionButtonSize, iconSize: actionIconSize),
                          _VideoActionButton(
                            onPressed: () async {
                              final confirmed = await confirmAction(context, title: "Delete video?", message: "Delete $videoLabel?", confirmLabel: "Delete", destructive: true);
                              if (confirmed) {
                                onDelete();
                              }
                            },
                            tooltip: "Delete",
                            icon: Icons.delete_rounded,
                            backgroundColor: theme.colorScheme.errorContainer,
                            foregroundColor: theme.colorScheme.onErrorContainer,
                            size: actionButtonSize,
                            iconSize: actionIconSize,
                          ),
                          if (onOpenChannel != null) _VideoActionButton(onPressed: onOpenChannel!, tooltip: "Open channel", icon: Icons.grid_view_rounded, foregroundColor: theme.colorScheme.primary, size: actionButtonSize, iconSize: actionIconSize),
                          _VideoActionButton(onPressed: onPlay, tooltip: "Play", icon: Icons.local_movies_rounded, foregroundColor: theme.colorScheme.primary, size: actionButtonSize, iconSize: actionIconSize),
                          _VideoActionButton(onPressed: onToggleBookmark, tooltip: "Bookmark", icon: (video.bookmark ?? false) ? Icons.favorite_rounded : Icons.favorite_border_rounded, foregroundColor: (video.bookmark ?? false) ? Colors.pink : theme.colorScheme.primary, size: actionButtonSize, iconSize: actionIconSize),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _VideoMetadataItem extends StatelessWidget {
  const _VideoMetadataItem({required this.icon, required this.iconSize, required this.label, this.style});

  final IconData icon;
  final double iconSize;
  final String label;
  final TextStyle? style;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Icon(icon, size: iconSize),
        const SizedBox(width: 4),
        Text(label, style: style),
      ],
    );
  }
}

class _VideoActionButton extends StatelessWidget {
  const _VideoActionButton({required this.onPressed, required this.tooltip, required this.icon, this.backgroundColor, this.foregroundColor, this.size = 34, this.iconSize = 18});

  final VoidCallback onPressed;
  final String tooltip;
  final IconData icon;
  final Color? backgroundColor;
  final Color? foregroundColor;
  final double size;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return RemoteFocusableAction(
      onPressed: onPressed,
      borderRadius: BorderRadius.circular(9),
      focusPadding: 1,
      scaleOnFocus: 1.05,
      child: Tooltip(
        message: tooltip,
        child: Material(
          color: backgroundColor ?? theme.colorScheme.primary.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(9),
          child: SizedBox(
            width: size,
            height: size,
            child: Icon(icon, size: iconSize, color: foregroundColor ?? theme.colorScheme.primary),
          ),
        ),
      ),
    );
  }
}
