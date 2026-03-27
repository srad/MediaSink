import "dart:async";

import "package:flutter/material.dart";

class InteractiveVideoPreview extends StatefulWidget {
  const InteractiveVideoPreview({
    super.key,
    required this.coverUrl,
    required this.frameUrls,
    required this.onTap,
    this.height = 180,
  });

  final String coverUrl;
  final List<String> frameUrls;
  final VoidCallback onTap;
  final double height;

  @override
  State<InteractiveVideoPreview> createState() => _InteractiveVideoPreviewState();
}

class _InteractiveVideoPreviewState extends State<InteractiveVideoPreview> {
  static const Duration _frameDuration = Duration(milliseconds: 250);

  Timer? _timer;
  var _isPreviewing = false;
  var _frameIndex = 0;

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  void _setPreviewing(bool isPreviewing) {
    if (_isPreviewing == isPreviewing) {
      return;
    }

    if (!mounted) {
      return;
    }

    if (widget.frameUrls.isEmpty) {
      setState(() {
        _isPreviewing = false;
        _frameIndex = 0;
      });
      return;
    }

    if (isPreviewing) {
      setState(() {
        _isPreviewing = true;
        _frameIndex = 0;
      });
      _timer?.cancel();
      _timer = Timer.periodic(_frameDuration, (_) {
        if (!mounted || !_isPreviewing || widget.frameUrls.isEmpty) {
          _timer?.cancel();
          return;
        }
        setState(() {
          _frameIndex = (_frameIndex + 1) % widget.frameUrls.length;
        });
      });
      return;
    }

    _timer?.cancel();
    setState(() {
      _isPreviewing = false;
      _frameIndex = 0;
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final imageUrl = _isPreviewing && widget.frameUrls.isNotEmpty ? widget.frameUrls[_frameIndex] : widget.coverUrl;
    final progress = widget.frameUrls.isEmpty ? 0.0 : (_frameIndex + 1) / widget.frameUrls.length;

    return MouseRegion(
      onEnter: (_) => _setPreviewing(true),
      onExit: (_) => _setPreviewing(false),
      child: GestureDetector(
        onTap: widget.onTap,
        onLongPressStart: (_) => _setPreviewing(true),
        onLongPressEnd: (_) => _setPreviewing(false),
        onLongPressCancel: () => _setPreviewing(false),
        child: Stack(
          alignment: Alignment.center,
          children: <Widget>[
            ClipRRect(
              borderRadius: const BorderRadius.vertical(top: Radius.circular(10)),
              child: SizedBox(
                width: double.infinity,
                height: widget.height,
                child: Image.network(
                  imageUrl,
                  fit: BoxFit.cover,
                  gaplessPlayback: true,
                  errorBuilder: (context, error, stackTrace) {
                    return ColoredBox(
                      color: theme.colorScheme.surfaceContainerHighest,
                      child: Center(
                        child: Icon(
                          Icons.image_not_supported_outlined,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    );
                  },
                ),
              ),
            ),
            const Icon(Icons.play_circle_fill, size: 64, color: Colors.white70),
            if (widget.frameUrls.isNotEmpty)
              Positioned(
                left: 0,
                right: 0,
                bottom: 0,
                child: AnimatedOpacity(
                  duration: const Duration(milliseconds: 120),
                  opacity: _isPreviewing ? 1 : 0,
                  child: Container(
                    height: 8,
                    color: Colors.white.withValues(alpha: 0.45),
                    child: FractionallySizedBox(
                      alignment: Alignment.centerLeft,
                      widthFactor: progress.clamp(0.0, 1.0),
                      child: Container(color: theme.colorScheme.primary.withValues(alpha: 0.8)),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
