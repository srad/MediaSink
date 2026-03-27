import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../formatters.dart";
import "../history_controller.dart";
import "../media_sink_api.dart";
import "../models.dart";
import "../widgets/inline_error_banner.dart";
import "../widgets/preview_frame.dart";
import "video_player_page.dart";

class HistoryScreen extends StatelessWidget {
  const HistoryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<HistoryController>();
    final api = context.read<MediaSinkApi>();

    if (controller.loading && controller.entries.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (controller.entries.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: const _HistoryEmptyState(),
          ),
        ),
      );
    }

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
      children: <Widget>[
        if (controller.error != null)
          Padding(
            padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
            child: InlineErrorBanner(message: controller.error!, onRetry: () => controller.reload()),
          ),
        for (final entry in controller.entries) ...<Widget>[
          _HistoryCard(
            entry: entry,
            previewUrl: api.previewUrl(entry.video),
            onOpen: () => _openHistoryEntry(context, controller, api, entry),
            onRemove: () => controller.removeEntry(entry),
          ),
          const SizedBox(height: 10),
        ],
      ],
    );
  }

  Future<void> _openHistoryEntry(
    BuildContext context,
    HistoryController controller,
    MediaSinkApi api,
    PlayedVideoHistoryEntry entry,
  ) async {
    try {
      final video = await controller.resolveVideo(entry);
      if (!context.mounted) {
        return;
      }

      final result = await Navigator.of(context).push<VideoPlayerResult>(
        MaterialPageRoute<VideoPlayerResult>(
          builder: (_) => VideoPlayerPage(
            title: video.filename,
            url: api.videoFileUrl(video),
            video: video,
          ),
        ),
      );

      if (result == VideoPlayerResult.deleted && context.mounted) {
        await controller.removeEntry(entry);
      }
    } catch (error) {
      if (!context.mounted) {
        return;
      }
      final message = error.toString();
      final text = message.toLowerCase().contains("404") || message.toLowerCase().contains("not found")
          ? "Video is no longer available. Removed from history."
          : message;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
    }
  }
}

class _HistoryEmptyState extends StatelessWidget {
  const _HistoryEmptyState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 28),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Container(
              width: 64,
              height: 64,
              decoration: BoxDecoration(
                color: theme.colorScheme.primary.withValues(alpha: 0.10),
                borderRadius: BorderRadius.circular(18),
              ),
              child: Icon(
                Icons.history_rounded,
                size: 34,
                color: theme.colorScheme.primary,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              "No played videos yet",
              textAlign: TextAlign.center,
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              "Videos you watch for a few seconds will appear here, so you can jump back in later.",
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HistoryCard extends StatelessWidget {
  const _HistoryCard({
    required this.entry,
    required this.previewUrl,
    required this.onOpen,
    required this.onRemove,
  });

  final PlayedVideoHistoryEntry entry;
  final String previewUrl;
  final VoidCallback onOpen;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: InkWell(
        onTap: onOpen,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.all(10),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              PreviewFrame(
                imageUrl: previewUrl,
                width: 128,
                height: 72,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    // Text(
                    //   entry.video.filename,
                    //   maxLines: 2,
                    //   overflow: TextOverflow.ellipsis,
                    //   style: theme.textTheme.titleSmall,
                    // ),
                    // const SizedBox(height: 4),
                    Text(
                      entry.video.channelName,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 10,
                      runSpacing: 6,
                      children: <Widget>[
                        // _HistoryMetaChip(
                        //   icon: Icons.history_rounded,
                        //   label: playedLabel,
                        // ),
                        _HistoryMetaChip(
                          icon: Icons.timer_rounded,
                          label: formatDuration(entry.video.duration),
                        ),
                        _HistoryMetaChip(
                          icon: Icons.sd_storage_rounded,
                          label: formatBytes(entry.video.size),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Column(
                children: <Widget>[
                  IconButton(
                    tooltip: "Play",
                    onPressed: onOpen,
                    icon: const Icon(Icons.play_arrow_rounded),
                  ),
                  IconButton(
                    tooltip: "Remove from history",
                    onPressed: onRemove,
                    icon: const Icon(Icons.history_toggle_off_rounded),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _HistoryMetaChip extends StatelessWidget {
  const _HistoryMetaChip({
    required this.icon,
    required this.label,
  });

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
      decoration: BoxDecoration(
        color: theme.colorScheme.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 14, color: theme.colorScheme.primary),
          const SizedBox(width: 5),
          Text(
            label,
            style: theme.textTheme.labelMedium,
          ),
        ],
      ),
    );
  }
}
