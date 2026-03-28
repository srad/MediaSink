import "package:flutter/material.dart";

class AdaptiveGridSpec {
  const AdaptiveGridSpec({required this.minItemWidth, required this.maxColumns});

  final double minItemWidth;
  final int maxColumns;
}

bool useExpandedGridColumns(BuildContext context) {
  return MediaQuery.sizeOf(context).shortestSide >= 600;
}

int adaptiveGridMaxColumns(BuildContext context, {int compact = 3, int expanded = 4}) {
  return useExpandedGridColumns(context) ? expanded : compact;
}

AdaptiveGridSpec mediaGridSpec(BuildContext context) {
  if (useExpandedGridColumns(context)) {
    return const AdaptiveGridSpec(minItemWidth: 280, maxColumns: 4);
  }
  return const AdaptiveGridSpec(minItemWidth: 240, maxColumns: 3);
}

AdaptiveGridSpec streamGridSpec(BuildContext context) {
  if (useExpandedGridColumns(context)) {
    return const AdaptiveGridSpec(minItemWidth: 280, maxColumns: 4);
  }
  return const AdaptiveGridSpec(minItemWidth: 280, maxColumns: 2);
}

AdaptiveGridSpec channelGridSpec(BuildContext context) {
  if (useExpandedGridColumns(context)) {
    return const AdaptiveGridSpec(minItemWidth: 260, maxColumns: 4);
  }
  return const AdaptiveGridSpec(minItemWidth: 220, maxColumns: 3);
}
