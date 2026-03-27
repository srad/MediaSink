import "package:flutter/material.dart";

class PreviewFrame extends StatelessWidget {
  const PreviewFrame({super.key, required this.imageUrl, this.height = 56, this.width = 96});

  final String imageUrl;
  final double height;
  final double width;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ClipRRect(
      borderRadius: BorderRadius.circular(6),
      child: Container(
        color: theme.colorScheme.surfaceContainerHighest,
        width: width,
        height: height,
        child: Image.network(
          imageUrl,
          fit: BoxFit.cover,
          errorBuilder: (context, error, stackTrace) {
            return Center(child: Icon(Icons.image_not_supported_outlined, color: theme.colorScheme.onSurfaceVariant));
          },
        ),
      ),
    );
  }
}
