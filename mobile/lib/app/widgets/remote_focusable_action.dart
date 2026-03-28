import "package:flutter/material.dart";
import "package:flutter/services.dart";

class RemoteFocusableAction extends StatefulWidget {
  const RemoteFocusableAction({super.key, required this.child, required this.onPressed, this.borderRadius = const BorderRadius.all(Radius.circular(10)), this.enabled = true, this.autofocus = false, this.onFocusChange, this.behavior = HitTestBehavior.deferToChild, this.focusPadding = 2, this.scaleOnFocus = 1.01, this.onLongPressStart, this.onLongPressEnd, this.onLongPressCancel});

  final Widget child;
  final VoidCallback onPressed;
  final BorderRadius borderRadius;
  final bool enabled;
  final bool autofocus;
  final ValueChanged<bool>? onFocusChange;
  final HitTestBehavior behavior;
  final double focusPadding;
  final double scaleOnFocus;
  final GestureLongPressStartCallback? onLongPressStart;
  final GestureLongPressEndCallback? onLongPressEnd;
  final VoidCallback? onLongPressCancel;

  @override
  State<RemoteFocusableAction> createState() => _RemoteFocusableActionState();
}

class _RemoteFocusableActionState extends State<RemoteFocusableAction> {
  var _isFocused = false;

  static const Map<ShortcutActivator, Intent> _shortcuts = <ShortcutActivator, Intent>{SingleActivator(LogicalKeyboardKey.select): ActivateIntent(), SingleActivator(LogicalKeyboardKey.enter): ActivateIntent(), SingleActivator(LogicalKeyboardKey.numpadEnter): ActivateIntent(), SingleActivator(LogicalKeyboardKey.space): ActivateIntent(), SingleActivator(LogicalKeyboardKey.gameButtonA): ActivateIntent()};

  void _handleFocusChange(bool isFocused) {
    if (_isFocused == isFocused) {
      return;
    }

    setState(() {
      _isFocused = isFocused;
    });

    widget.onFocusChange?.call(isFocused);

    if (isFocused) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        Scrollable.ensureVisible(context, duration: const Duration(milliseconds: 160), curve: Curves.easeOut, alignmentPolicy: ScrollPositionAlignmentPolicy.keepVisibleAtEnd);
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final focusColor = widget.enabled ? theme.colorScheme.secondary : theme.disabledColor;

    return Semantics(
      button: true,
      enabled: widget.enabled,
      child: FocusableActionDetector(
        enabled: widget.enabled,
        autofocus: widget.autofocus,
        shortcuts: _shortcuts,
        actions: <Type, Action<Intent>>{
          ActivateIntent: CallbackAction<ActivateIntent>(
            onInvoke: (_) {
              if (widget.enabled) {
                widget.onPressed();
              }
              return null;
            },
          ),
        },
        onFocusChange: _handleFocusChange,
        child: Padding(
          padding: EdgeInsets.all(widget.focusPadding),
          child: AnimatedScale(
            scale: _isFocused ? widget.scaleOnFocus : 1,
            duration: const Duration(milliseconds: 120),
            curve: Curves.easeOut,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 120),
              decoration: BoxDecoration(
                borderRadius: widget.borderRadius,
                border: _isFocused ? Border.all(color: focusColor, width: 3) : null,
                boxShadow: _isFocused ? <BoxShadow>[BoxShadow(color: focusColor.withValues(alpha: 0.35), blurRadius: 18, spreadRadius: 1)] : null,
              ),
              child: GestureDetector(
                behavior: widget.behavior,
                onTap: widget.enabled ? widget.onPressed : null,
                onLongPressStart: widget.enabled ? widget.onLongPressStart : null,
                onLongPressEnd: widget.enabled ? widget.onLongPressEnd : null,
                onLongPressCancel: widget.enabled ? widget.onLongPressCancel : null,
                child: ClipRRect(borderRadius: widget.borderRadius, child: widget.child),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
