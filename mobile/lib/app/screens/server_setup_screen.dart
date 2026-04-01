import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:provider/provider.dart";

import "../models.dart";
import "../session_controller.dart";

class ServerSetupScreen extends StatefulWidget {
  const ServerSetupScreen({super.key});

  @override
  State<ServerSetupScreen> createState() => _ServerSetupScreenState();
}

class _ServerSetupScreenState extends State<ServerSetupScreen> {
  late final TextEditingController _originController;
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();
  final FocusNode _originFocusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    _originController = TextEditingController();
    // Fire TV / Android TV may not show the keyboard on programmatic focus;
    // explicitly request it whenever the field gains focus.
    _originFocusNode.addListener(() {
      if (_originFocusNode.hasFocus) {
        SystemChannels.textInput.invokeMethod<void>("TextInput.show");
      }
    });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final session = context.read<AppSessionController>();
    if (_originController.text.isEmpty && session.savedOrigin != null) {
      _originController.text = session.savedOrigin!;
    }
  }

  @override
  void dispose() {
    _originFocusNode.dispose();
    _originController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }

    await context.read<AppSessionController>().configureServer(_originController.text.trim());
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<AppSessionController>();
    final theme = Theme.of(context);
    final isBusy = session.status == SessionStatus.booting;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 520),
              child: Card(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Form(
                    key: _formKey,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        Text("Connect to MediaSink", style: theme.textTheme.headlineSmall),
                        const SizedBox(height: 8),
                        Text("Enter the base server URL. The app will derive API and WebSocket endpoints from it and verify the server API version before login.", style: theme.textTheme.bodyMedium),
                        const SizedBox(height: 24),
                        TextFormField(
                          controller: _originController,
                          focusNode: _originFocusNode,
                          autofocus: true,
                          decoration: const InputDecoration(labelText: "Server URL", hintText: "http://192.168.1.50:3000", border: OutlineInputBorder()),
                          validator: (value) {
                            final trimmed = value?.trim() ?? "";
                            final uri = Uri.tryParse(trimmed);
                            if (trimmed.isEmpty) {
                              return "Server URL is required.";
                            }
                            if (uri == null || !uri.isAbsolute) {
                              return "Enter a valid absolute URL.";
                            }
                            return null;
                          },
                          textInputAction: TextInputAction.done,
                          onFieldSubmitted: (_) => _submit(),
                        ),
                        if ((session.message ?? "").isNotEmpty) ...<Widget>[const SizedBox(height: 16), Text(session.message!, style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.error))],
                        if (session.config != null) ...<Widget>[
                          const SizedBox(height: 20),
                          Container(
                            width: double.infinity,
                            padding: const EdgeInsets.all(16),
                            decoration: BoxDecoration(color: theme.colorScheme.surfaceContainerHighest, borderRadius: BorderRadius.circular(16)),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: <Widget>[
                                Text("Detected server", style: theme.textTheme.titleMedium),
                                const SizedBox(height: 8),
                                Text("Version: ${session.config!.appVersion.isEmpty ? "unknown" : session.config!.appVersion}"),
                                Text("Build: ${session.config!.build.isEmpty ? "unknown" : session.config!.build}"),
                                Text("API: ${session.config!.apiVersion.isEmpty ? "unknown" : session.config!.apiVersion}"),
                              ],
                            ),
                          ),
                        ],
                        const SizedBox(height: 24),
                        SizedBox(
                          width: double.infinity,
                          child: FilledButton(
                            onPressed: isBusy ? null : _submit,
                            child: isBusy ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Text("Verify server"),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
