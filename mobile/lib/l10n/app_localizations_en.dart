// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class SEn extends S {
  SEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'MediaSink';

  @override
  String get helloWorld => 'Hello Modern World!';

  @override
  String welcomeMessage(String userName) {
    return 'Welcome $userName to our app!';
  }
}
