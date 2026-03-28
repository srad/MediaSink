import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../library_controller.dart";
import "../formatters.dart";
import "../grid_layout.dart";
import "../media_sink_api.dart";
import "../models.dart";
import "../widgets/classic_video_card.dart";
import "../widgets/inline_error_banner.dart";
import "../widgets/responsive_card_grid.dart";
import "channel_detail_screen.dart";
import "video_player_page.dart";
import "../../api/export.dart";

class LibraryScreen extends StatelessWidget {
  const LibraryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<LibraryController>();
    final api = context.read<MediaSinkApi>();
    final spec = mediaGridSpec(context);
    final videos = controller.videos;

    return RefreshIndicator(
      onRefresh: controller.refresh,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(4, 4, 4, 12),
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: SegmentedButton<LibrarySection>(
              segments: const <ButtonSegment<LibrarySection>>[
                ButtonSegment<LibrarySection>(value: LibrarySection.latest, label: Text("Latest"), icon: Icon(Icons.schedule)),
                ButtonSegment<LibrarySection>(value: LibrarySection.bookmarks, label: Text("Bookmarks"), icon: Icon(Icons.favorite_border)),
                ButtonSegment<LibrarySection>(value: LibrarySection.random, label: Text("Random"), icon: Icon(Icons.shuffle)),
              ],
              selected: <LibrarySection>{controller.section},
              onSelectionChanged: (selection) => controller.setSection(selection.first),
            ),
          ),
          const SizedBox(height: 8),
          _FiltersLauncherCard(controller: controller),
          _ResultsSummaryCard(controller: controller),
          if (controller.loading && videos.isNotEmpty) const Padding(padding: EdgeInsets.fromLTRB(12, 0, 12, 10), child: LinearProgressIndicator()),
          if (controller.loading && videos.isEmpty)
            const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
          if (controller.error != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
              child: InlineErrorBanner(message: controller.error!, onRetry: controller.refresh),
            ),
          if (!controller.loading && videos.isEmpty) const Padding(padding: EdgeInsets.all(16), child: Text("No videos available for this view.")),
          if (videos.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
              child: ResponsiveCardGrid(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                minItemWidth: spec.minItemWidth,
                maxColumns: spec.maxColumns,
                mainAxisExtent: 268,
                crossAxisSpacing: 12,
                mainAxisSpacing: 12,
                itemCount: videos.length,
                itemBuilder: (context, index) {
                  final video = videos[index];
                  return ClassicVideoCard(
                    video: video,
                    previewUrl: api.previewUrl(video),
                    previewFrames: api.previewFrames(video),
                    onOpen: () async => _openVideo(context, controller, api, video),
                    onPlay: () async => _openVideo(context, controller, api, video),
                    onDownload: () async {
                      final path = await api.downloadVideo(video);
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text("Saved to $path")));
                      }
                    },
                    onDelete: () async {
                      await controller.deleteVideo(video);
                    },
                    onToggleBookmark: () async {
                      await controller.toggleBookmark(video);
                    },
                    onOpenChannel: video.channelId == null
                        ? null
                        : () {
                            Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: video.channelId!)));
                          },
                  );
                },
              ),
            ),
          if (controller.section == LibrarySection.latest && controller.totalPages > 1) _PaginationCard(controller: controller),
        ],
      ),
    );
  }

  Future<void> _openVideo(BuildContext context, LibraryController controller, MediaSinkApi api, DbRecording video) async {
    final result = await Navigator.of(context).push<VideoPlayerResult>(
      MaterialPageRoute<VideoPlayerResult>(
        builder: (_) => VideoPlayerPage(title: video.filename, url: api.videoFileUrl(video), video: video),
      ),
    );
    if (result == VideoPlayerResult.deleted && context.mounted) {
      await controller.refresh();
    }
  }
}

class _FiltersLauncherCard extends StatelessWidget {
  const _FiltersLauncherCard({required this.controller});

  final LibraryController controller;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 0, 8, 8),
      child: Card(
        child: ListTile(
          dense: true,
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
          leading: const Icon(Icons.tune_rounded),
          title: Text(switch (controller.section) {
            LibrarySection.latest => "Latest filters",
            LibrarySection.bookmarks => "Bookmark filter",
            LibrarySection.random => "Random limit",
          }, style: theme.textTheme.titleSmall),
          subtitle: Text(_filterSummary(controller)),
          trailing: Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              if (controller.section == LibrarySection.random) IconButton(onPressed: controller.loading ? null : controller.refresh, icon: const Icon(Icons.refresh_rounded), tooltip: "Refresh random videos"),
              FilledButton.tonalIcon(onPressed: controller.loading ? null : () => _openFilters(context, controller), icon: const Icon(Icons.tune_rounded), label: const Text("Filters")),
            ],
          ),
        ),
      ),
    );
  }
}

String _sortColumnLabel(RequestsVideoSortColumn value) {
  return switch (value) {
    RequestsVideoSortColumn.createdAt => "Created",
    RequestsVideoSortColumn.size => "Filesize",
    RequestsVideoSortColumn.duration => "Duration",
    RequestsVideoSortColumn.$unknown => "Unknown",
  };
}

String _sortOrderLabel(ModelsSortOrder value) {
  return switch (value) {
    ModelsSortOrder.asc => "Ascending",
    ModelsSortOrder.desc => "Descending",
    ModelsSortOrder.$unknown => "Unknown",
  };
}

String _filterSummary(LibraryController controller) {
  return switch (controller.section) {
    LibrarySection.latest => "${_sortColumnLabel(controller.sortColumn)} • ${_sortOrderLabel(controller.sortOrder)} • ${controller.pageSize}/page",
    LibrarySection.bookmarks => controller.bookmarkChannelFilter.isEmpty ? "All channels" : controller.bookmarkChannelFilter,
    LibrarySection.random => "${controller.randomLimit} videos",
  };
}

Future<void> _openFilters(BuildContext context, LibraryController controller) {
  return switch (controller.section) {
    LibrarySection.latest => _showLatestFiltersSheet(context, controller),
    LibrarySection.bookmarks => _showBookmarksFiltersSheet(context, controller),
    LibrarySection.random => _showRandomFiltersSheet(context, controller),
  };
}

Future<void> _showLatestFiltersSheet(BuildContext context, LibraryController controller) async {
  var draftSortColumn = controller.sortColumn;
  var draftSortOrder = controller.sortOrder;
  var draftPageSize = controller.pageSize;

  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (context) {
      return StatefulBuilder(
        builder: (context, setModalState) {
          final mediaQuery = MediaQuery.of(context);
          final bottomInset = mediaQuery.viewInsets.bottom > 0 ? mediaQuery.viewInsets.bottom : mediaQuery.viewPadding.bottom;

          return SafeArea(
            top: false,
            bottom: false,
            child: SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(16, 4, 16, bottomInset + 16),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  Text("Video filters", style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 4),
                  Text("Choose sorting and page size for the latest videos view.", style: Theme.of(context).textTheme.bodySmall),
                  const SizedBox(height: 16),
                  DropdownButtonFormField<RequestsVideoSortColumn>(
                    initialValue: draftSortColumn,
                    decoration: const InputDecoration(border: OutlineInputBorder(), labelText: "Order by"),
                    items: const <DropdownMenuItem<RequestsVideoSortColumn>>[
                      DropdownMenuItem(value: RequestsVideoSortColumn.createdAt, child: Text("Created at")),
                      DropdownMenuItem(value: RequestsVideoSortColumn.size, child: Text("Filesize")),
                      DropdownMenuItem(value: RequestsVideoSortColumn.duration, child: Text("Duration")),
                    ],
                    onChanged: (value) {
                      if (value != null) {
                        setModalState(() => draftSortColumn = value);
                      }
                    },
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<ModelsSortOrder>(
                    initialValue: draftSortOrder,
                    decoration: const InputDecoration(border: OutlineInputBorder(), labelText: "Order"),
                    items: const <DropdownMenuItem<ModelsSortOrder>>[
                      DropdownMenuItem(value: ModelsSortOrder.asc, child: Text("Ascending")),
                      DropdownMenuItem(value: ModelsSortOrder.desc, child: Text("Descending")),
                    ],
                    onChanged: (value) {
                      if (value != null) {
                        setModalState(() => draftSortOrder = value);
                      }
                    },
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<int>(
                    initialValue: draftPageSize,
                    decoration: const InputDecoration(border: OutlineInputBorder(), labelText: "Limit"),
                    items: LibraryController.pageSizeOptions.map((value) => DropdownMenuItem<int>(value: value, child: Text(value.toString()))).toList(growable: false),
                    onChanged: (value) {
                      if (value != null) {
                        setModalState(() => draftPageSize = value);
                      }
                    },
                  ),
                  const SizedBox(height: 18),
                  Row(
                    children: <Widget>[
                      OutlinedButton(
                        onPressed: () {
                          setModalState(() {
                            draftSortColumn = RequestsVideoSortColumn.createdAt;
                            draftSortOrder = ModelsSortOrder.desc;
                            draftPageSize = LibraryController.defaultPageSize;
                          });
                        },
                        child: const Text("Reset"),
                      ),
                      const Spacer(),
                      FilledButton(
                        onPressed: () async {
                          final navigator = Navigator.of(context);
                          navigator.pop();
                          await controller.applyLatestFilters(sortColumn: draftSortColumn, sortOrder: draftSortOrder, pageSize: draftPageSize);
                        },
                        child: const Text("Apply"),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          );
        },
      );
    },
  );
}

Future<void> _showBookmarksFiltersSheet(BuildContext context, LibraryController controller) async {
  var draftChannel = controller.bookmarkChannelFilter;

  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (context) {
      return StatefulBuilder(
        builder: (context, setModalState) {
          final mediaQuery = MediaQuery.of(context);
          final bottomInset = mediaQuery.viewInsets.bottom > 0 ? mediaQuery.viewInsets.bottom : mediaQuery.viewPadding.bottom;

          return SafeArea(
            top: false,
            bottom: false,
            child: SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(16, 4, 16, bottomInset + 16),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  Text("Bookmark filter", style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 4),
                  Text("Limit bookmarked videos to a single channel or show them all.", style: Theme.of(context).textTheme.bodySmall),
                  const SizedBox(height: 16),
                  DropdownButtonFormField<String>(
                    initialValue: draftChannel,
                    decoration: const InputDecoration(border: OutlineInputBorder(), labelText: "Channel"),
                    items: <DropdownMenuItem<String>>[
                      const DropdownMenuItem<String>(value: "", child: Text("Show all")),
                      ...controller.bookmarkChannels.map((channel) => DropdownMenuItem<String>(value: channel, child: Text(channel))),
                    ],
                    onChanged: (value) {
                      setModalState(() => draftChannel = value ?? "");
                    },
                  ),
                  const SizedBox(height: 18),
                  Row(
                    children: <Widget>[
                      OutlinedButton(
                        onPressed: () {
                          setModalState(() => draftChannel = "");
                        },
                        child: const Text("Reset"),
                      ),
                      const Spacer(),
                      FilledButton(
                        onPressed: () {
                          Navigator.of(context).pop();
                          controller.setBookmarkChannelFilter(draftChannel);
                        },
                        child: const Text("Apply"),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          );
        },
      );
    },
  );
}

Future<void> _showRandomFiltersSheet(BuildContext context, LibraryController controller) async {
  var draftLimit = controller.randomLimit;

  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (context) {
      return StatefulBuilder(
        builder: (context, setModalState) {
          final mediaQuery = MediaQuery.of(context);
          final bottomInset = mediaQuery.viewInsets.bottom > 0 ? mediaQuery.viewInsets.bottom : mediaQuery.viewPadding.bottom;

          return SafeArea(
            top: false,
            bottom: false,
            child: SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(16, 4, 16, bottomInset + 16),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  Text("Random videos", style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 4),
                  Text("Choose how many random videos to load each time.", style: Theme.of(context).textTheme.bodySmall),
                  const SizedBox(height: 16),
                  DropdownButtonFormField<int>(
                    initialValue: draftLimit,
                    decoration: const InputDecoration(border: OutlineInputBorder(), labelText: "Limit"),
                    items: LibraryController.pageSizeOptions.map((value) => DropdownMenuItem<int>(value: value, child: Text(value.toString()))).toList(growable: false),
                    onChanged: (value) {
                      if (value != null) {
                        setModalState(() => draftLimit = value);
                      }
                    },
                  ),
                  const SizedBox(height: 18),
                  Row(
                    children: <Widget>[
                      OutlinedButton(
                        onPressed: () {
                          setModalState(() => draftLimit = LibraryController.defaultRandomLimit);
                        },
                        child: const Text("Reset"),
                      ),
                      const Spacer(),
                      FilledButton(
                        onPressed: () async {
                          final navigator = Navigator.of(context);
                          navigator.pop();
                          await controller.setRandomLimit(draftLimit);
                        },
                        child: const Text("Apply"),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          );
        },
      );
    },
  );
}

class _ResultsSummaryCard extends StatelessWidget {
  const _ResultsSummaryCard({required this.controller});

  final LibraryController controller;

  @override
  Widget build(BuildContext context) {
    final label = switch (controller.section) {
      LibrarySection.latest => controller.totalCount == 0 ? "No filtered videos" : "Showing ${controller.visibleStart}-${controller.visibleEnd} of ${controller.totalCount} videos",
      LibrarySection.bookmarks => controller.videos.isEmpty ? "No bookmarked videos" : "${controller.videos.length} bookmarked video${controller.videos.length == 1 ? "" : "s"}",
      LibrarySection.random => controller.videos.isEmpty ? "No random videos" : "${controller.videos.length} random video${controller.videos.length == 1 ? "" : "s"}",
    };

    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
      child: Card(
        child: ListTile(
          dense: true,
          leading: Icon(switch (controller.section) {
            LibrarySection.latest => Icons.filter_alt_rounded,
            LibrarySection.bookmarks => Icons.favorite_rounded,
            LibrarySection.random => Icons.shuffle_rounded,
          }),
          title: Text(label),
          subtitle: controller.section == LibrarySection.latest && controller.totalCount > 0 ? Text("Page ${controller.currentPage} / ${controller.totalPages} • ${formatBytes(controller.videos.fold<int>(0, (sum, video) => sum + (video.size ?? 0)))} on this page") : null,
        ),
      ),
    );
  }
}

class _PaginationCard extends StatelessWidget {
  const _PaginationCard({required this.controller});

  final LibraryController controller;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 2, 8, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Row(
            children: <Widget>[
              SizedBox(
                width: 84,
                child: _PagerNavButton(label: "Prev", icon: Icons.chevron_left_rounded, trailingIcon: false, enabled: !controller.loading && controller.canGoToPreviousPage, onPressed: controller.previousPage),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Center(
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        for (final page in controller.visiblePageItems)
                          if (page == null)
                            const Padding(padding: EdgeInsets.symmetric(horizontal: 4), child: Text("..."))
                          else
                            Padding(
                              padding: const EdgeInsets.symmetric(horizontal: 3),
                              child: _PagerPageButton(label: page.toString(), selected: page == controller.currentPage, enabled: !controller.loading, onPressed: () => controller.goToPage(page)),
                            ),
                      ],
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: 84,
                child: _PagerNavButton(label: "Next", icon: Icons.chevron_right_rounded, trailingIcon: true, enabled: !controller.loading && controller.canGoToNextPage, onPressed: controller.nextPage),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text("Showing ${controller.visibleStart}-${controller.visibleEnd} of ${controller.totalCount}", textAlign: TextAlign.center, style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
    );
  }
}

class _PagerNavButton extends StatelessWidget {
  const _PagerNavButton({required this.label, required this.icon, required this.trailingIcon, required this.enabled, required this.onPressed});

  final String label;
  final IconData icon;
  final bool trailingIcon;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FilledButton.tonal(
      onPressed: enabled ? onPressed : null,
      style: FilledButton.styleFrom(minimumSize: const Size(84, 36), padding: const EdgeInsets.symmetric(horizontal: 10), tapTargetSize: MaterialTapTargetSize.shrinkWrap, visualDensity: VisualDensity.compact),
      child: Row(mainAxisAlignment: MainAxisAlignment.center, children: trailingIcon ? <Widget>[Text(label), const SizedBox(width: 4), Icon(icon, size: 18)] : <Widget>[Icon(icon, size: 18), const SizedBox(width: 4), Text(label)]),
    );
  }
}

class _PagerPageButton extends StatelessWidget {
  const _PagerPageButton({required this.label, required this.selected, required this.enabled, required this.onPressed});

  final String label;
  final bool selected;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final style = OutlinedButton.styleFrom(minimumSize: const Size(40, 36), padding: const EdgeInsets.symmetric(horizontal: 12), tapTargetSize: MaterialTapTargetSize.shrinkWrap, visualDensity: VisualDensity.compact);

    if (selected) {
      return FilledButton.tonal(onPressed: enabled ? onPressed : null, style: style, child: Text(label));
    }

    return OutlinedButton(onPressed: enabled ? onPressed : null, style: style, child: Text(label));
  }
}
