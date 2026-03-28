import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:flutter_test/flutter_test.dart";
import "package:mediasink_app/app/widgets/remote_focusable_action.dart";

void main() {
  testWidgets("remote focusable action triggers activate intent from remote select", (tester) async {
    var pressed = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: RemoteFocusableAction(
            autofocus: true,
            onPressed: () {
              pressed += 1;
            },
            child: const SizedBox(width: 120, height: 60, child: ColoredBox(color: Colors.blue)),
          ),
        ),
      ),
    );
    await tester.pump();

    await tester.sendKeyEvent(LogicalKeyboardKey.select);
    await tester.pump();

    expect(pressed, 1);
  });
}
