import "package:flutter/material.dart";

import "../../api/export.dart";
import "preview_frame.dart";

class ClassicChannelTile extends StatelessWidget {
  const ClassicChannelTile({super.key, required this.channel, required this.previewUrl, required this.onTap});

  final ServicesChannelInfo channel;
  final String previewUrl;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Stack(
        children: <Widget>[
          Positioned.fill(
            child: ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: PreviewFrame(imageUrl: previewUrl, width: double.infinity, height: double.infinity),
            ),
          ),
          Positioned.fill(
            child: Container(
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(8),
                gradient: LinearGradient(colors: <Color>[Colors.black.withValues(alpha: 0.6), Colors.transparent], begin: Alignment.bottomCenter, end: Alignment.topCenter),
              ),
            ),
          ),
          Positioned(
            bottom: 8,
            left: 8,
            right: 8,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  channel.displayName ?? channel.channelName ?? "No name",
                  style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                Row(
                  children: <Widget>[
                    if (channel.fav ?? false) const Icon(Icons.favorite, color: Colors.pink, size: 16),
                    if (channel.isOnline ?? false) const Icon(Icons.circle, color: Colors.green, size: 12),
                    if (channel.isPaused ?? false) const Icon(Icons.pause_circle_filled, color: Colors.orange, size: 16),
                    if (channel.isRecording ?? false) const Icon(Icons.fiber_manual_record, color: Colors.red, size: 16),
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
