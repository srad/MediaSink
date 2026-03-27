// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

@JsonEnum()
enum RequestsEstimateEnhancementRequestEncodingPreset {
  @JsonValue('veryfast')
  veryfast('veryfast'),
  @JsonValue('faster')
  faster('faster'),
  @JsonValue('fast')
  fast('fast'),
  @JsonValue('medium')
  medium('medium'),
  @JsonValue('slow')
  slow('slow'),
  @JsonValue('slower')
  slower('slower'),
  @JsonValue('veryslow')
  veryslow('veryslow'),
  /// Default value for all unparsed values, allows backward compatibility when adding new values on the backend.
  $unknown(null);

  const RequestsEstimateEnhancementRequestEncodingPreset(this.json);

  factory RequestsEstimateEnhancementRequestEncodingPreset.fromJson(String json) => values.firstWhere(
        (e) => e.json == json,
        orElse: () => $unknown,
      );

  final String? json;

  @override
  String toString() => json?.toString() ?? super.toString();
  /// Returns all defined enum values excluding the $unknown value.
  static List<RequestsEstimateEnhancementRequestEncodingPreset> get $valuesDefined => values.where((value) => value != $unknown).toList();
}
