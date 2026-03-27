// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

@JsonEnum()
enum DatabaseJobStatus {
  @JsonValue('completed')
  completed('completed'),
  @JsonValue('open')
  open('open'),
  @JsonValue('error')
  error('error'),
  @JsonValue('canceled')
  canceled('canceled'),
  /// Default value for all unparsed values, allows backward compatibility when adding new values on the backend.
  $unknown(null);

  const DatabaseJobStatus(this.json);

  factory DatabaseJobStatus.fromJson(String json) => values.firstWhere(
        (e) => e.json == json,
        orElse: () => $unknown,
      );

  final String? json;
}
