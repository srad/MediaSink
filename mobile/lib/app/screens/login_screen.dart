import "package:flutter/material.dart";
import "package:provider/provider.dart";

import "../models.dart";
import "../session_controller.dart";

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  late final TextEditingController _usernameController;
  late final TextEditingController _passwordController;
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();

  @override
  void initState() {
    super.initState();
    _usernameController = TextEditingController();
    _passwordController = TextEditingController();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final session = context.read<AppSessionController>();
    if (_usernameController.text.isEmpty && session.savedUsername != null) {
      _usernameController.text = session.savedUsername!;
    }
  }

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }

    await context.read<AppSessionController>().login(username: _usernameController.text.trim(), password: _passwordController.text);
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<AppSessionController>();
    final theme = Theme.of(context);
    final isBusy = session.status == SessionStatus.authenticating;

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
                        Text("Sign in", style: theme.textTheme.headlineSmall),
                        const SizedBox(height: 8),
                        Text(session.savedOrigin ?? "Server not configured", style: theme.textTheme.bodyMedium),
                        const SizedBox(height: 12),
                        TextButton.icon(onPressed: isBusy ? null : () => session.resetServer(), icon: const Icon(Icons.settings_ethernet), label: const Text("Change server")),
                        const SizedBox(height: 12),
                        TextFormField(
                          controller: _usernameController,
                          decoration: const InputDecoration(labelText: "Username", border: OutlineInputBorder()),
                          validator: (value) => (value == null || value.trim().isEmpty) ? "Username is required." : null,
                          textInputAction: TextInputAction.next,
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _passwordController,
                          decoration: const InputDecoration(labelText: "Password", border: OutlineInputBorder()),
                          obscureText: true,
                          validator: (value) => (value == null || value.isEmpty) ? "Password is required." : null,
                          textInputAction: TextInputAction.done,
                          onFieldSubmitted: (_) => _submit(),
                        ),
                        if ((session.message ?? "").isNotEmpty) ...<Widget>[const SizedBox(height: 16), Text(session.message!, style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.error))],
                        const SizedBox(height: 24),
                        SizedBox(
                          width: double.infinity,
                          child: FilledButton(
                            onPressed: isBusy ? null : _submit,
                            child: isBusy ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Text("Sign in"),
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
