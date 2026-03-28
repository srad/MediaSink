import "dart:math" as math;

import "package:flutter/material.dart";

class ResponsiveCardGrid extends StatelessWidget {
  const ResponsiveCardGrid({super.key, required this.itemCount, required this.itemBuilder, this.minItemWidth = 320, this.maxColumns, this.padding = EdgeInsets.zero, this.mainAxisSpacing = 10, this.crossAxisSpacing = 10, this.mainAxisExtent, this.childAspectRatio, this.physics, this.shrinkWrap = false}) : assert(mainAxisExtent != null || childAspectRatio != null, "Either mainAxisExtent or childAspectRatio must be provided.");

  final int itemCount;
  final IndexedWidgetBuilder itemBuilder;
  final double minItemWidth;
  final int? maxColumns;
  final EdgeInsetsGeometry padding;
  final double mainAxisSpacing;
  final double crossAxisSpacing;
  final double? mainAxisExtent;
  final double? childAspectRatio;
  final ScrollPhysics? physics;
  final bool shrinkWrap;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final availableWidth = constraints.maxWidth.isFinite ? constraints.maxWidth : MediaQuery.sizeOf(context).width;
        var crossAxisCount = math.max(1, ((availableWidth + crossAxisSpacing) / (minItemWidth + crossAxisSpacing)).floor());
        if (maxColumns != null) {
          crossAxisCount = math.min(crossAxisCount, maxColumns!);
        }

        return GridView.builder(
          shrinkWrap: shrinkWrap,
          physics: physics,
          padding: padding,
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: crossAxisCount, mainAxisSpacing: mainAxisSpacing, crossAxisSpacing: crossAxisSpacing, mainAxisExtent: mainAxisExtent, childAspectRatio: childAspectRatio ?? 1),
          itemCount: itemCount,
          itemBuilder: itemBuilder,
        );
      },
    );
  }
}
