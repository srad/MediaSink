import "package:flutter/material.dart";

Future<bool> confirmAction(BuildContext context, {required String title, required String message, String confirmLabel = "Confirm", String cancelLabel = "Cancel", bool destructive = false}) async {
  final confirmed = await showDialog<bool>(
    context: context,
    builder: (dialogContext) {
      return AlertDialog(
        title: Text(title),
        content: Text(message),
        actions: <Widget>[
          TextButton(onPressed: () => Navigator.of(dialogContext).pop(false), child: Text(cancelLabel)),
          FilledButton(
            style: destructive ? FilledButton.styleFrom(backgroundColor: Theme.of(dialogContext).colorScheme.error, foregroundColor: Theme.of(dialogContext).colorScheme.onError) : null,
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(confirmLabel),
          ),
        ],
      );
    },
  );

  return confirmed ?? false;
}
