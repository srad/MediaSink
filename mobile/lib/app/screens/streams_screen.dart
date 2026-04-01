import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:provider/provider.dart";

import "../../api/export.dart";
import "../channels_controller.dart";
import "../grid_layout.dart";
import "../media_sink_api.dart";
import "../widgets/classic_stream_card.dart";
import "../widgets/inline_error_banner.dart";
import "../widgets/responsive_card_grid.dart";
import "channel_detail_screen.dart";
import "channel_editor_sheet.dart";

class StreamsScreen extends StatefulWidget {
  const StreamsScreen({super.key});

  @override
  State<StreamsScreen> createState() => _StreamsScreenState();
}

class _StreamsScreenState extends State<StreamsScreen> with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  late final TextEditingController _searchController;
  final FocusNode _searchFocusNode = FocusNode();
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _searchController = TextEditingController();
    _searchFocusNode.addListener(() {
      if (_searchFocusNode.hasFocus) {
        SystemChannels.textInput.invokeMethod<void>("TextInput.show");
      }
    });
    _tabController.addListener(_handleTabChanged);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_initialized) {
      return;
    }
    final controller = context.read<ChannelsController>();
    _searchController.text = controller.streamSearchQuery;
    _tabController.index = controller.streamTabIndex.clamp(0, 2);
    _initialized = true;
  }

  @override
  void dispose() {
    _tabController
      ..removeListener(_handleTabChanged)
      ..dispose();
    _searchController.dispose();
    _searchFocusNode.dispose();
    super.dispose();
  }

  void _handleTabChanged() {
    if (_tabController.indexIsChanging) {
      return;
    }
    context.read<ChannelsController>().setStreamTabIndex(_tabController.index);
  }

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<ChannelsController>();
    final api = context.read<MediaSinkApi>();
    final onlineChannels = controller.streamOnlineChannels;
    final offlineChannels = controller.streamOfflineChannels;
    final disabledChannels = controller.streamDisabledChannels;

    if (controller.loading && controller.channels.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    return Column(
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.fromLTRB(8, 8, 8, 6),
          child: Row(
            children: <Widget>[
              Expanded(
                child: TextField(
                  controller: _searchController,
                  focusNode: _searchFocusNode,
                  onChanged: controller.setStreamSearchQuery,
                  decoration: InputDecoration(
                    hintText: "search... #tag",
                    prefixIcon: const Icon(Icons.search_rounded),
                    suffixIcon: controller.streamSearchQuery.isEmpty
                        ? null
                        : IconButton(
                            onPressed: () {
                              _searchController.clear();
                              controller.setStreamSearchQuery("");
                            },
                            icon: const Icon(Icons.close_rounded),
                          ),
                    isDense: true,
                    border: const OutlineInputBorder(),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              IconButton.filledTonal(onPressed: () => controller.setStreamFavoritesOnly(!controller.streamFavoritesOnly), tooltip: controller.streamFavoritesOnly ? "Show all streams" : "Show favorites only", icon: Icon(controller.streamFavoritesOnly ? Icons.favorite_rounded : Icons.favorite_border_rounded)),
            ],
          ),
        ),
        TabBar(
          controller: _tabController,
          tabs: <Tab>[
            Tab(
              text: "Online (${onlineChannels.length})",
              icon: const _StreamTabIcon(icon: Icons.fiber_manual_record_rounded, color: Colors.redAccent),
            ),
            Tab(
              text: "Offline (${offlineChannels.length})",
              icon: const _StreamTabIcon(icon: Icons.portable_wifi_off_rounded, color: Colors.blueGrey),
            ),
            Tab(
              text: "Disabled (${disabledChannels.length})",
              icon: const _StreamTabIcon(icon: Icons.do_not_disturb_alt_rounded, color: Colors.amber),
            ),
          ],
        ),
        if (controller.error != null)
          Padding(
            padding: const EdgeInsets.fromLTRB(8, 10, 8, 0),
            child: InlineErrorBanner(message: controller.error!, onRetry: controller.refresh),
          ),
        Expanded(
          child: TabBarView(
            controller: _tabController,
            children: <Widget>[
              _ChannelTabList(
                channels: onlineChannels,
                api: api,
                onRefresh: controller.refresh,
                onOpenDetails: (channelId) {
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: channelId)));
                },
                onEdit: (id) => openChannelEditorFlow(context, id: id),
                onDelete: controller.deleteChannel,
                onTogglePause: controller.togglePause,
                onToggleFavorite: controller.toggleFavorite,
              ),
              _ChannelTabList(
                channels: offlineChannels,
                api: api,
                onRefresh: controller.refresh,
                onOpenDetails: (channelId) {
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: channelId)));
                },
                onEdit: (id) => openChannelEditorFlow(context, id: id),
                onDelete: controller.deleteChannel,
                onTogglePause: controller.togglePause,
                onToggleFavorite: controller.toggleFavorite,
              ),
              _ChannelTabList(
                channels: disabledChannels,
                api: api,
                onRefresh: controller.refresh,
                onOpenDetails: (channelId) {
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: channelId)));
                },
                onEdit: (id) => openChannelEditorFlow(context, id: id),
                onDelete: controller.deleteChannel,
                onTogglePause: controller.togglePause,
                onToggleFavorite: controller.toggleFavorite,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _StreamTabIcon extends StatelessWidget {
  const _StreamTabIcon({required this.icon, required this.color});

  final IconData icon;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 26,
      height: 26,
      decoration: BoxDecoration(color: color.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(999)),
      child: Icon(icon, size: 16, color: color),
    );
  }
}

class _ChannelTabList extends StatelessWidget {
  const _ChannelTabList({required this.channels, required this.api, required this.onRefresh, required this.onOpenDetails, required this.onEdit, required this.onDelete, required this.onTogglePause, required this.onToggleFavorite});

  final List<ServicesChannelInfo> channels;
  final MediaSinkApi api;
  final Future<void> Function() onRefresh;
  final void Function(int channelId) onOpenDetails;
  final Future<void> Function(int? id) onEdit;
  final Future<void> Function(int id) onDelete;
  final Future<void> Function(ServicesChannelInfo channel) onTogglePause;
  final Future<void> Function(ServicesChannelInfo channel) onToggleFavorite;

  @override
  Widget build(BuildContext context) {
    final spec = streamGridSpec(context);

    if (channels.isEmpty) {
      return RefreshIndicator(
        onRefresh: onRefresh,
        child: ListView(
          children: const <Widget>[
            SizedBox(height: 120),
            Center(child: Text("No channels match this view.")),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ResponsiveCardGrid(
        minItemWidth: spec.minItemWidth,
        maxColumns: spec.maxColumns,
        mainAxisExtent: 320,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 8),
        itemCount: channels.length,
        crossAxisSpacing: 12,
        mainAxisSpacing: 12,
        itemBuilder: (context, index) {
          final channel = channels[index];
          return ClassicStreamCard(channel: channel, previewUrl: "${api.config.fileBaseUrl}/${channel.preview ?? ""}", onOpenDetails: () => onOpenDetails(channel.channelId!), onEdit: () => onEdit(channel.channelId), onDelete: () => onDelete(channel.channelId!), onTogglePause: () => onTogglePause(channel), onToggleFavorite: () => onToggleFavorite(channel));
        },
      ),
    );
  }
}
