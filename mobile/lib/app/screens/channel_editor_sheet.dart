import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../../api/export.dart";
import "../action_confirmation.dart";
import "../channels_controller.dart";

const _channelEditorSheetAnimationStyle = AnimationStyle(duration: Duration(milliseconds: 320), reverseDuration: Duration(milliseconds: 280));

Future<RequestsChannelRequest?> showChannelEditorSheet(BuildContext context, {ServicesChannelInfo? initial}) {
  return showModalBottomSheet<RequestsChannelRequest>(
    context: context,
    isScrollControlled: true,
    sheetAnimationStyle: _channelEditorSheetAnimationStyle,
    builder: (_) => ChannelEditorSheet(initial: initial),
  );
}

Future<void> openChannelEditorFlow(BuildContext context, {int? id}) async {
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

class ChannelEditorSheet extends StatefulWidget {
  const ChannelEditorSheet({super.key, this.initial});

  final ServicesChannelInfo? initial;

  @override
  State<ChannelEditorSheet> createState() => _ChannelEditorSheetState();
}

class _ChannelEditorSheetState extends State<ChannelEditorSheet> {
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();

  late final TextEditingController _channelNameController;
  late final TextEditingController _displayNameController;
  late final TextEditingController _urlController;
  late final TextEditingController _minDurationController;
  late final TextEditingController _skipStartController;
  late final TextEditingController _tagsController;
  bool _paused = false;

  @override
  void initState() {
    super.initState();
    final initial = widget.initial;
    _channelNameController = TextEditingController(text: initial?.channelName ?? "");
    _displayNameController = TextEditingController(text: initial?.displayName ?? "");
    _urlController = TextEditingController(text: initial?.url ?? "");
    _minDurationController = TextEditingController(text: (initial?.minDuration ?? 0).toString());
    _skipStartController = TextEditingController(text: (initial?.skipStart ?? 0).toString());
    _tagsController = TextEditingController(text: (initial?.tags ?? const <String>[]).join(", "));
    _paused = initial?.isPaused ?? false;
  }

  @override
  void dispose() {
    _channelNameController.dispose();
    _displayNameController.dispose();
    _urlController.dispose();
    _minDurationController.dispose();
    _skipStartController.dispose();
    _tagsController.dispose();
    super.dispose();
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }

    Navigator.of(context).pop(RequestsChannelRequest(channelName: _channelNameController.text.trim(), displayName: _displayNameController.text.trim(), url: _urlController.text.trim(), minDuration: int.tryParse(_minDurationController.text.trim()) ?? 0, skipStart: int.tryParse(_skipStartController.text.trim()) ?? 0, isPaused: _paused, deleted: false, fav: widget.initial?.fav ?? false, tags: _tagsController.text.split(",").map((item) => item.trim()).where((item) => item.isNotEmpty).toList()));
  }

  @override
  Widget build(BuildContext context) {
    final mediaQuery = MediaQuery.of(context);
    final bottomInset = mediaQuery.viewInsets.bottom > 0 ? mediaQuery.viewInsets.bottom : mediaQuery.viewPadding.bottom;
    const verticalMargin = 16.0;

    return SafeArea(
      top: false,
      bottom: false,
      child: AnimatedPadding(
        duration: const Duration(milliseconds: 150),
        curve: Curves.easeOut,
        padding: EdgeInsets.only(left: 16, right: 16, top: verticalMargin, bottom: bottomInset + verticalMargin),
        child: SingleChildScrollView(
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                Text(widget.initial == null ? "Add channel" : "Edit channel", style: Theme.of(context).textTheme.titleLarge),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _channelNameController,
                  decoration: const InputDecoration(labelText: "Channel name", border: OutlineInputBorder()),
                  validator: (value) => (value == null || value.trim().isEmpty) ? "Required" : null,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _displayNameController,
                  decoration: const InputDecoration(labelText: "Display name", border: OutlineInputBorder()),
                  validator: (value) => (value == null || value.trim().isEmpty) ? "Required" : null,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _urlController,
                  decoration: const InputDecoration(labelText: "Stream URL", border: OutlineInputBorder()),
                  validator: (value) => (value == null || value.trim().isEmpty) ? "Required" : null,
                ),
                const SizedBox(height: 12),
                Row(
                  children: <Widget>[
                    Expanded(
                      child: TextFormField(
                        controller: _minDurationController,
                        decoration: const InputDecoration(labelText: "Min duration", border: OutlineInputBorder()),
                        keyboardType: TextInputType.number,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: TextFormField(
                        controller: _skipStartController,
                        decoration: const InputDecoration(labelText: "Skip start", border: OutlineInputBorder()),
                        keyboardType: TextInputType.number,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _tagsController,
                  decoration: const InputDecoration(labelText: "Tags", helperText: "Comma-separated", border: OutlineInputBorder()),
                ),
                SwitchListTile(title: const Text("Start paused"), value: _paused, onChanged: (value) => setState(() => _paused = value)),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(onPressed: _submit, child: const Text("Save")),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
