// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

@JsonEnum()
enum DbJobPriority {
  @JsonValue(1)
  value1(1),
  @JsonValue(3)
  value3(3),
  @JsonValue(5)
  value5(5),
  /// Default value for all unparsed values, allows backward compatibility when adding new values on the backend.
  $unknown(null);

  const DbJobPriority(this.json);

  factory DbJobPriority.fromJson(int json) => values.firstWhere(
        (e) => e.json == json,
        orElse: () => $unknown,
      );

  final int? json;

  @override
  String toString() => json?.toString() ?? super.toString();
  /// Returns all defined enum values excluding the $unknown value.
  static List<DbJobPriority> get $valuesDefined => values.where((value) => value != $unknown).toList();
}
