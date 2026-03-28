import "package:flutter/material.dart";
import "package:provider/provider.dart";
import "package:url_launcher/url_launcher.dart";

import "../../api/export.dart";
import "../action_confirmation.dart";
import "../channels_controller.dart";
import "../formatters.dart";
import "../grid_layout.dart";
import "../media_sink_api.dart";
import "../library_controller.dart";
import "../widgets/classic_video_card.dart";
import "../widgets/preview_frame.dart";
import "../widgets/responsive_card_grid.dart";
import "channel_editor_sheet.dart";
import "video_player_page.dart";

class ChannelDetailScreen extends StatefulWidget {
  const ChannelDetailScreen({super.key, required this.channelId});

  final int channelId;

  @override
  State<ChannelDetailScreen> createState() => _ChannelDetailScreenState();
}

class _ChannelDetailScreenState extends State<ChannelDetailScreen> {
  late Future<ServicesChannelInfo> _future;

  @override
  void initState() {
    super.initState();
    _future = context.read<ChannelsController>().loadChannel(widget.channelId);
  }

  Future<void> _reload() async {
    final controller = context.read<ChannelsController>();
    setState(() {
      _future = controller.loadChannel(widget.channelId);
    });
    await _future;
    await controller.refresh();
  }

  @override
  Widget build(BuildContext context) {
    final controller = context.read<ChannelsController>();
    final api = context.read<MediaSinkApi>();
    final library = context.read<LibraryController>();
    final spec = mediaGridSpec(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text("Channel"),
        actions: <Widget>[IconButton(onPressed: _reload, icon: const Icon(Icons.refresh))],
      ),
      body: SafeArea(
        top: false,
        child: FutureBuilder<ServicesChannelInfo>(
          future: _future,
          builder: (context, snapshot) {
            if (!snapshot.hasData) {
              if (snapshot.hasError) {
                return Center(child: Text(snapshot.error.toString()));
              }
              return const Center(child: CircularProgressIndicator());
            }

            final channel = snapshot.data!;
            final recordings = channel.recordings ?? const <DbRecording>[];
            final previewPath = channel.preview ?? "";

            return RefreshIndicator(
              onRefresh: _reload,
              child: ListView(
                padding: const EdgeInsets.only(top: 8, bottom: 16),
                children: <Widget>[
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                    child: _ChannelHeroCard(
                      channel: channel,
                      previewUrl: "${api.config.fileBaseUrl}/$previewPath",
                      recordingsCount: channel.recordingsCount ?? recordings.length,
                      recordingsSize: channel.recordingsSize,
                      onOpenExternal: () => _openExternal(channel.url),
                      onTogglePause: () async {
                        final confirmed = await confirmAction(context, title: (channel.isPaused ?? false) ? "Resume channel?" : "Pause channel?", message: "${(channel.isPaused ?? false) ? "Resume" : "Pause"} ${channel.displayName ?? channel.channelName ?? "this channel"}?", confirmLabel: (channel.isPaused ?? false) ? "Resume" : "Pause");
                        if (!confirmed) {
                          return;
                        }
                        await controller.togglePause(channel);
                        await _reload();
                      },
                      onToggleFavorite: () async {
                        await controller.toggleFavorite(channel);
                        await _reload();
                      },
                      onEdit: () async {
                        final request = await showChannelEditorSheet(context, initial: channel);
                        if (request != null) {
                          if (!context.mounted) {
                            return;
                          }
                          final confirmed = await confirmAction(context, title: "Save channel changes?", message: "Save changes for ${request.displayName}?", confirmLabel: "Save");
                          if (!confirmed) {
                            return;
                          }
                          await controller.saveChannel(id: channel.channelId, request: request);
                          await _reload();
                        }
                      },
                      onDelete: () async {
                        final confirmed = await confirmAction(context, title: "Delete channel?", message: "Delete ${channel.displayName ?? channel.channelName ?? "this channel"}?", confirmLabel: "Delete", destructive: true);
                        if (!confirmed) {
                          return;
                        }
                        await controller.deleteChannel(channel.channelId!);
                        if (context.mounted) {
                          Navigator.of(context).pop();
                        }
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(8, 14, 8, 8),
                    child: Row(
                      children: <Widget>[
                        Text("Videos", style: Theme.of(context).textTheme.titleMedium),
                        const SizedBox(width: 8),
                        Text("${recordings.length}", style: Theme.of(context).textTheme.bodySmall),
                      ],
                    ),
                  ),
                  if (recordings.isEmpty)
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      child: Card(
                        child: Padding(
                          padding: const EdgeInsets.all(16),
                          child: Text("No videos for this channel yet.", style: Theme.of(context).textTheme.bodyMedium),
                        ),
                      ),
                    )
                  else
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      child: ResponsiveCardGrid(
                        shrinkWrap: true,
                        physics: const NeverScrollableScrollPhysics(),
                        minItemWidth: spec.minItemWidth,
                        maxColumns: spec.maxColumns,
                        mainAxisExtent: 268,
                        crossAxisSpacing: 12,
                        mainAxisSpacing: 12,
                        itemCount: recordings.length,
                        itemBuilder: (context, index) {
                          final video = recordings[index];
                          return ClassicVideoCard(
                            video: video,
                            previewUrl: api.previewUrl(video),
                            previewFrames: api.previewFrames(video),
                            onOpen: () {
                              Navigator.of(context)
                                  .push<VideoPlayerResult>(
                                    MaterialPageRoute<VideoPlayerResult>(
                                      builder: (_) => VideoPlayerPage(title: video.filename, url: api.videoFileUrl(video), video: video),
                                    ),
                                  )
                                  .then((result) async {
                                    if (result == VideoPlayerResult.deleted && context.mounted) {
                                      await _reload();
                                    }
                                  });
                            },
                            onPlay: () {
                              Navigator.of(context)
                                  .push<VideoPlayerResult>(
                                    MaterialPageRoute<VideoPlayerResult>(
                                      builder: (_) => VideoPlayerPage(title: video.filename, url: api.videoFileUrl(video), video: video),
                                    ),
                                  )
                                  .then((result) async {
                                    if (result == VideoPlayerResult.deleted && context.mounted) {
                                      await _reload();
                                    }
                                  });
                            },
                            onDownload: () async {
                              final path = await api.downloadVideo(video);
                              if (context.mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text("Saved to $path")));
                              }
                            },
                            onDelete: () async {
                              await library.deleteVideo(video);
                              if (context.mounted) {
                                await _reload();
                              }
                            },
                            onToggleBookmark: () async {
                              await library.toggleBookmark(video);
                              if (context.mounted) {
                                await _reload();
                              }
                            },
                          );
                        },
                      ),
                    ),
                ],
              ),
            );
          },
        ),
      ),
    );
  }

  Future<void> _openExternal(String? rawUrl) async {
    if (rawUrl == null || rawUrl.isEmpty) {
      return;
    }
    final uri = Uri.tryParse(rawUrl);
    if (uri == null) {
      return;
    }
    await launchUrl(uri);
  }
}

class _ChannelHeroCard extends StatelessWidget {
  const _ChannelHeroCard({required this.channel, required this.previewUrl, required this.recordingsCount, required this.recordingsSize, required this.onOpenExternal, required this.onTogglePause, required this.onToggleFavorite, required this.onEdit, required this.onDelete});

  final ServicesChannelInfo channel;
  final String previewUrl;
  final int recordingsCount;
  final int? recordingsSize;
  final VoidCallback onOpenExternal;
  final VoidCallback onTogglePause;
  final VoidCallback onToggleFavorite;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final tags = channel.tags ?? const <String>[];
    final title = channel.displayName ?? channel.channelName ?? "Channel";
    final subtitle = channel.channelName ?? "";

    return Card(
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Stack(
            children: <Widget>[
              PreviewFrame(imageUrl: previewUrl, width: double.infinity, height: 172),
              Positioned.fill(
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    gradient: LinearGradient(begin: Alignment.topCenter, end: Alignment.bottomCenter, colors: <Color>[Colors.black.withValues(alpha: 0.06), Colors.black.withValues(alpha: 0.18), Colors.black.withValues(alpha: 0.52)]),
                  ),
                ),
              ),
              Positioned(
                top: 10,
                left: 10,
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: <Widget>[
                    _StatusPill(
                      icon: (channel.isPaused ?? false)
                          ? Icons.do_not_disturb_alt_rounded
                          : (channel.isOnline ?? false)
                          ? Icons.radio_button_checked_rounded
                          : Icons.portable_wifi_off_rounded,
                      label: (channel.isPaused ?? false)
                          ? "Disabled"
                          : (channel.isOnline ?? false)
                          ? "Online"
                          : "Offline",
                      color: (channel.isPaused ?? false)
                          ? Colors.amber
                          : (channel.isOnline ?? false)
                          ? Colors.redAccent
                          : Colors.blueGrey,
                    ),
                    if (channel.fav ?? false) const _StatusPill(icon: Icons.favorite_rounded, label: "Favorite", color: Colors.pinkAccent),
                  ],
                ),
              ),
              Positioned(
                left: 12,
                right: 12,
                bottom: 12,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      title,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.titleLarge?.copyWith(
                        color: Colors.white,
                        fontWeight: FontWeight.w700,
                        shadows: const <Shadow>[Shadow(offset: Offset(0, 1), blurRadius: 3, color: Colors.black45)],
                      ),
                    ),
                    if (subtitle.isNotEmpty && subtitle != title) ...<Widget>[
                      const SizedBox(height: 4),
                      Text(
                        subtitle,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodyMedium?.copyWith(color: Colors.white70),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
          Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                if ((channel.url ?? "").isNotEmpty)
                  InkWell(
                    onTap: onOpenExternal,
                    borderRadius: BorderRadius.circular(10),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
                      decoration: BoxDecoration(color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.35), borderRadius: BorderRadius.circular(10)),
                      child: Row(
                        children: <Widget>[
                          Icon(Icons.link_rounded, size: 18, color: theme.colorScheme.primary),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(channel.url!, maxLines: 1, overflow: TextOverflow.ellipsis, style: theme.textTheme.bodyMedium),
                          ),
                          const SizedBox(width: 6),
                          const Icon(Icons.open_in_new_rounded, size: 16),
                        ],
                      ),
                    ),
                  ),
                const SizedBox(height: 10),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: <Widget>[
                    _InfoPill(icon: Icons.videocam_rounded, label: "$recordingsCount videos"),
                    _InfoPill(icon: Icons.sd_storage_rounded, label: formatBytes(recordingsSize)),
                  ],
                ),
                if (tags.isNotEmpty) ...<Widget>[const SizedBox(height: 10), Wrap(spacing: 8, runSpacing: 8, children: tags.map((tag) => _TagPill(label: tag)).toList(growable: false))],
                const SizedBox(height: 12),
                Row(
                  children: <Widget>[
                    _HeaderActionButton(onPressed: onTogglePause, tooltip: (channel.isPaused ?? false) ? "Resume" : "Pause", icon: (channel.isPaused ?? false) ? Icons.play_arrow_rounded : Icons.pause_rounded),
                    const SizedBox(width: 8),
                    _HeaderActionButton(onPressed: onToggleFavorite, tooltip: "Favorite", icon: (channel.fav ?? false) ? Icons.favorite_rounded : Icons.favorite_border_rounded, foregroundColor: (channel.fav ?? false) ? Colors.pink : null),
                    const SizedBox(width: 8),
                    _HeaderActionButton(onPressed: onEdit, tooltip: "Edit", icon: Icons.edit_rounded),
                    const SizedBox(width: 8),
                    _HeaderActionButton(onPressed: onDelete, tooltip: "Delete", icon: Icons.delete_outline_rounded, backgroundColor: theme.colorScheme.errorContainer, foregroundColor: theme.colorScheme.onErrorContainer),
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

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.icon, required this.label, required this.color});

  final IconData icon;
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(color: Colors.black.withValues(alpha: 0.36), borderRadius: BorderRadius.circular(999)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 14, color: color),
          const SizedBox(width: 6),
          Text(
            label,
            style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 12),
          ),
        ],
      ),
    );
  }
}

class _InfoPill extends StatelessWidget {
  const _InfoPill({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(color: theme.colorScheme.primary.withValues(alpha: 0.08), borderRadius: BorderRadius.circular(999)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 14, color: theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text(label, style: theme.textTheme.labelMedium),
        ],
      ),
    );
  }
}

class _TagPill extends StatelessWidget {
  const _TagPill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(color: Theme.of(context).colorScheme.surfaceContainerHighest.withValues(alpha: 0.45), borderRadius: BorderRadius.circular(999)),
      child: Text(label, style: Theme.of(context).textTheme.labelMedium),
    );
  }
}

class _HeaderActionButton extends StatelessWidget {
  const _HeaderActionButton({required this.onPressed, required this.tooltip, required this.icon, this.backgroundColor, this.foregroundColor});

  final VoidCallback onPressed;
  final String tooltip;
  final IconData icon;
  final Color? backgroundColor;
  final Color? foregroundColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: backgroundColor ?? theme.colorScheme.primary.withValues(alpha: 0.10),
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(10),
        child: Tooltip(
          message: tooltip,
          child: SizedBox(width: 40, height: 40, child: Icon(icon, size: 20, color: foregroundColor ?? theme.colorScheme.primary)),
        ),
      ),
    );
  }
}
