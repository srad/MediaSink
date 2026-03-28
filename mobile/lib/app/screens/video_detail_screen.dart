import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../../api/export.dart";
import "../action_confirmation.dart";
import "../formatters.dart";
import "../library_controller.dart";
import "../media_sink_api.dart";
import "../widgets/preview_frame.dart";
import "video_player_page.dart";

class VideoDetailScreen extends StatelessWidget {
  const VideoDetailScreen({super.key, required this.video});

  final DbRecording video;

  @override
  Widget build(BuildContext context) {
    final api = context.read<MediaSinkApi>();
    final library = context.read<LibraryController>();

    return Scaffold(
      appBar: AppBar(
        title: Text(video.filename),
        actions: <Widget>[
          IconButton(
            onPressed: () async {
              await library.toggleBookmark(video);
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text("Bookmark updated.")));
              }
            },
            icon: Icon((video.bookmark ?? false) ? Icons.favorite : Icons.favorite_border),
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.only(top: 12, bottom: 24),
          children: <Widget>[
            Center(
              child: PreviewFrame(imageUrl: api.previewUrl(video), width: MediaQuery.of(context).size.width - 32, height: 220),
            ),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(video.channelName, style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    Text("Created: ${formatDateTime(video.createdAt)}"),
                    Text("Duration: ${formatDuration(video.duration)}"),
                    Text("Size: ${formatBytes(video.size)}"),
                    Text("Resolution: ${video.width ?? 0}x${video.height ?? 0}"),
                  ],
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Wrap(
                spacing: 12,
                runSpacing: 12,
                children: <Widget>[
                  FilledButton.icon(
                    onPressed: () async {
                      final result = await Navigator.of(context).push<VideoPlayerResult>(
                        MaterialPageRoute<VideoPlayerResult>(
                          builder: (_) => VideoPlayerPage(title: video.filename, url: api.videoFileUrl(video), video: video),
                        ),
                      );
                      if (result == VideoPlayerResult.deleted && context.mounted) {
                        Navigator.of(context).pop();
                      }
                    },
                    icon: const Icon(Icons.play_arrow),
                    label: const Text("Play"),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: () async {
                      final path = await api.downloadVideo(video);
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text("Saved to $path")));
                      }
                    },
                    icon: const Icon(Icons.download),
                    label: const Text("Download"),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: () async {
                      final confirmed = await confirmAction(context, title: "Refresh preview?", message: "Queue preview regeneration for ${video.filename}?", confirmLabel: "Queue");
                      if (!confirmed) {
                        return;
                      }
                      await library.refreshPreview(video);
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text("Preview regeneration queued.")));
                      }
                    },
                    icon: const Icon(Icons.refresh),
                    label: const Text("Refresh preview"),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: () async {
                      final confirmed = await confirmAction(context, title: "Delete video?", message: "Delete ${video.filename}?", confirmLabel: "Delete", destructive: true);
                      if (!confirmed) {
                        return;
                      }
                      await library.deleteVideo(video);
                      if (context.mounted) {
                        Navigator.of(context).pop();
                      }
                    },
                    icon: const Icon(Icons.delete_outline),
                    label: const Text("Delete"),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
