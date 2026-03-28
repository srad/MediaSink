import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../action_confirmation.dart";
import "../channels_controller.dart";
import "../error_utils.dart";
import "../grid_layout.dart";
import "../media_sink_api.dart";
import "../models.dart";
import "../widgets/classic_channel_list_tile.dart";
import "../widgets/classic_channel_tile.dart";
import "../widgets/inline_error_banner.dart";
import "../widgets/responsive_card_grid.dart";
import "channel_detail_screen.dart";
import "channel_editor_sheet.dart";

String _channelsSortLabel(ChannelsSortField field) {
  return switch (field) {
    ChannelsSortField.recording => "Recording",
    ChannelsSortField.name => "Name",
    ChannelsSortField.favorite => "Favorite",
    ChannelsSortField.videos => "Videos",
    ChannelsSortField.size => "Size",
    ChannelsSortField.added => "Added",
  };
}

class ChannelsScreen extends StatefulWidget {
  const ChannelsScreen({super.key, this.embedded = false});

  final bool embedded;

  @override
  State<ChannelsScreen> createState() => _ChannelsScreenState();
}

class _ChannelsScreenState extends State<ChannelsScreen> {
  late final TextEditingController _searchController;
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    _searchController = TextEditingController();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_initialized) {
      return;
    }
    _searchController.text = context.read<ChannelsController>().channelsSearchQuery;
    _initialized = true;
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _openEditor(BuildContext context, {int? id}) async {
    final controller = context.read<ChannelsController>();
    final initial = id == null ? null : controller.channelById(id);
    final request = await showChannelEditorSheet(context, initial: initial);

    if (request != null) {
      if (!context.mounted) {
        return;
      }
      final confirmed = await confirmAction(context, title: id == null ? "Create channel?" : "Save channel changes?", message: id == null ? "Create channel \"${request.displayName}\"?" : "Save changes for channel \"${request.displayName}\"?", confirmLabel: id == null ? "Create" : "Save");
      if (!confirmed) {
        return;
      }
      await controller.saveChannel(id: id, request: request);
    }
  }

  Future<void> _importChannels(BuildContext context, ChannelsController controller) async {
    final confirmed = await confirmAction(context, title: "Import channels?", message: "Choose a JSON file and import its channels into this server.", confirmLabel: "Import");
    if (!confirmed) {
      return;
    }

    try {
      final result = await controller.importChannelsFromFile();
      if (!context.mounted || result.cancelled) {
        return;
      }
      final messenger = ScaffoldMessenger.of(context);
      if (result.hasSuccesses && !result.hasFailures) {
        messenger.showSnackBar(SnackBar(content: Text("Imported ${result.successCount} channel${result.successCount == 1 ? "" : "s"}.")));
      } else if (result.hasSuccesses && result.hasFailures) {
        messenger.showSnackBar(SnackBar(content: Text("Imported ${result.successCount}, ${result.failureCount} failed.")));
      } else {
        messenger.showSnackBar(const SnackBar(content: Text("No channels were imported.")));
      }
    } catch (error) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(friendlyErrorMessage(error))));
      }
    }
  }

  Future<void> _exportChannels(BuildContext context, ChannelsController controller) async {
    final confirmed = await confirmAction(context, title: "Export channels?", message: "Export the current channel list to a JSON file on this device.", confirmLabel: "Export");
    if (!confirmed) {
      return;
    }

    try {
      final path = await controller.exportChannelsJson();
      if (!context.mounted || path == null) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text("Exported channels to $path")));
    } catch (error) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(friendlyErrorMessage(error))));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<ChannelsController>();
    final api = context.read<MediaSinkApi>();
    final spec = channelGridSpec(context);
    final channels = controller.filteredChannels;

    if (controller.loading && controller.channels.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    final content = RefreshIndicator(
      onRefresh: controller.refresh,
      child: ListView(
        padding: const EdgeInsets.only(top: 8, bottom: 16),
        children: <Widget>[
          if (controller.error != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
              child: InlineErrorBanner(message: controller.error!, onRetry: controller.refresh),
            ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Row(
              children: <Widget>[
                Expanded(
                  child: TextField(
                    controller: _searchController,
                    onChanged: controller.setChannelsSearchQuery,
                    decoration: InputDecoration(
                      hintText: "search... #tag",
                      prefixIcon: const Icon(Icons.search_rounded),
                      suffixIcon: controller.channelsSearchQuery.isEmpty
                          ? null
                          : IconButton(
                              onPressed: () {
                                _searchController.clear();
                                controller.setChannelsSearchQuery("");
                              },
                              icon: const Icon(Icons.close_rounded),
                            ),
                      isDense: true,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton.filledTonal(onPressed: () => controller.setChannelsFavoritesOnly(!controller.channelsFavoritesOnly), tooltip: controller.channelsFavoritesOnly ? "Show all channels" : "Show favorites only", icon: Icon(controller.channelsFavoritesOnly ? Icons.favorite_rounded : Icons.favorite_border_rounded)),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(8, 8, 8, 10),
            child: Wrap(
              spacing: 8,
              runSpacing: 8,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                SegmentedButton<ChannelListLayout>(
                  segments: const <ButtonSegment<ChannelListLayout>>[
                    ButtonSegment<ChannelListLayout>(value: ChannelListLayout.grid, icon: Icon(Icons.grid_view_rounded), label: Text("Grid")),
                    ButtonSegment<ChannelListLayout>(value: ChannelListLayout.list, icon: Icon(Icons.view_list_rounded), label: Text("List")),
                  ],
                  selected: <ChannelListLayout>{controller.channelsLayout},
                  onSelectionChanged: (selection) => controller.setChannelsLayout(selection.first),
                ),
                PopupMenuButton<ChannelsSortField>(
                  initialValue: controller.channelsSortField,
                  onSelected: controller.setChannelsSortField,
                  itemBuilder: (context) => ChannelsSortField.values.map((field) => PopupMenuItem<ChannelsSortField>(value: field, child: Text(_channelsSortLabel(field)))).toList(growable: false),
                  child: IgnorePointer(
                    child: FilledButton.tonalIcon(onPressed: () {}, icon: const Icon(Icons.sort_rounded), label: Text(_channelsSortLabel(controller.channelsSortField))),
                  ),
                ),
                IconButton.filledTonal(onPressed: () => controller.setChannelsSortDescending(!controller.channelsSortDescending), tooltip: controller.channelsSortDescending ? "Sort descending" : "Sort ascending", icon: Icon(controller.channelsSortDescending ? Icons.arrow_downward_rounded : Icons.arrow_upward_rounded)),
                PopupMenuButton<String>(
                  enabled: !controller.isImportExportBusy,
                  onSelected: (value) {
                    if (value == "export") {
                      _exportChannels(context, controller);
                    } else if (value == "import") {
                      _importChannels(context, controller);
                    }
                  },
                  itemBuilder: (context) => const <PopupMenuEntry<String>>[PopupMenuItem<String>(value: "export", child: Text("Export channels")), PopupMenuItem<String>(value: "import", child: Text("Import channels"))],
                  child: IgnorePointer(
                    child: FilledButton.tonalIcon(
                      onPressed: () {},
                      icon: controller.isImportExportBusy ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.import_export_rounded),
                      label: const Text("Import/Export"),
                    ),
                  ),
                ),
                FilledButton.tonalIcon(onPressed: () => _openEditor(context), icon: const Icon(Icons.add_rounded), label: const Text("Add channel")),
              ],
            ),
          ),
          if (!controller.loading && channels.isEmpty) const Padding(padding: EdgeInsets.all(16), child: Text("No channels match the current filter.")),
          if (controller.channelsLayout == ChannelListLayout.grid)
            ResponsiveCardGrid(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              padding: const EdgeInsets.all(8),
              minItemWidth: spec.minItemWidth,
              maxColumns: spec.maxColumns,
              childAspectRatio: 3 / 2,
              crossAxisSpacing: 8,
              mainAxisSpacing: 8,
              itemCount: channels.length,
              itemBuilder: (context, index) {
                final channel = channels[index];
                return ClassicChannelTile(
                  channel: channel,
                  previewUrl: "${api.config.fileBaseUrl}/${channel.preview ?? ""}",
                  onTap: () {
                    Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: channel.channelId!)));
                  },
                );
              },
            )
          else
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              padding: const EdgeInsets.symmetric(horizontal: 8),
              itemBuilder: (context, index) {
                final channel = channels[index];
                return ClassicChannelListTile(
                  channel: channel,
                  previewUrl: "${api.config.fileBaseUrl}/${channel.preview ?? ""}",
                  onTap: () {
                    Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: channel.channelId!)));
                  },
                );
              },
              separatorBuilder: (_, _) => const SizedBox(height: 10),
              itemCount: channels.length,
            ),
        ],
      ),
    );

    if (widget.embedded) {
      return content;
    }

    return Scaffold(
      appBar: AppBar(title: const Text("Channels")),
      body: SafeArea(top: false, child: content),
    );
  }
}
