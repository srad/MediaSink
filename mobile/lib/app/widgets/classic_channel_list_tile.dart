import "package:flutter/material.dart";

import "../../api/export.dart";
import "../formatters.dart";
import "preview_frame.dart";

class ClassicChannelListTile extends StatelessWidget {
  const ClassicChannelListTile({
    super.key,
    required this.channel,
    required this.previewUrl,
    required this.onTap,
  });

  final ServicesChannelInfo channel;
  final String previewUrl;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = channel.displayName ?? channel.channelName ?? "No name";
    final videosCount = channel.recordingsCount ?? channel.recordings?.length ?? 0;
    final videosLabel = "$videosCount video${videosCount == 1 ? "" : "s"}";
    return Card(
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.all(10),
          child: Row(
            children: <Widget>[
              ClipRRect(
                borderRadius: BorderRadius.circular(8),
                child: PreviewFrame(imageUrl: previewUrl, width: 120, height: 68),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      displayName,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.titleSmall,
                    ),
                    const SizedBox(height: 6),
                    Wrap(
                      spacing: 8,
                      runSpacing: 6,
                      children: <Widget>[
                        _MetaChip(
                          icon: Icons.videocam_rounded,
                          label: videosLabel,
                          color: theme.colorScheme.primary,
                        ),
                        _MetaChip(
                          icon: Icons.sd_storage_rounded,
                          label: formatBytes(channel.recordingsSize),
                          color: theme.colorScheme.primary,
                        ),
                        _StatusDot(
                          color: (channel.isPaused ?? false)
                              ? Colors.amber
                              : (channel.isOnline ?? false)
                                  ? Colors.redAccent
                                  : Colors.blueGrey,
                          label: (channel.isPaused ?? false)
                              ? "Disabled"
                              : (channel.isOnline ?? false)
                                  ? "Online"
                                  : "Offline",
                        ),
                        if (channel.fav ?? false) const _StatusDot(color: Colors.pink, label: "Favorite"),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MetaChip extends StatelessWidget {
  const _MetaChip({
    required this.icon,
    required this.label,
    required this.color,
  });

  final IconData icon;
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 12, color: color),
          const SizedBox(width: 6),
          Text(label, style: Theme.of(context).textTheme.labelSmall),
        ],
      ),
    );
  }
}

class _StatusDot extends StatelessWidget {
  const _StatusDot({required this.color, required this.label});

  final Color color;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(Icons.circle, size: 8, color: color),
          const SizedBox(width: 6),
          Text(label, style: Theme.of(context).textTheme.labelSmall),
        ],
      ),
    );
  }
}
