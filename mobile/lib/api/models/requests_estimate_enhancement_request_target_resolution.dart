// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

@JsonEnum()
enum RequestsEstimateEnhancementRequestTargetResolution {
  @JsonValue('720p')
  value720p('720p'),
  @JsonValue('1080p')
  value1080p('1080p'),
  @JsonValue('1440p')
  value1440p('1440p'),
  @JsonValue('4k')
  value4k('4k'),
  /// Default value for all unparsed values, allows backward compatibility when adding new values on the backend.
  $unknown(null);

  const RequestsEstimateEnhancementRequestTargetResolution(this.json);

  factory RequestsEstimateEnhancementRequestTargetResolution.fromJson(String json) => values.firstWhere(
        (e) => e.json == json,
        orElse: () => $unknown,
      );

  final String? json;

  @override
  String toString() => json?.toString() ?? super.toString();
  /// Returns all defined enum values excluding the $unknown value.
  static List<RequestsEstimateEnhancementRequestTargetResolution> get $valuesDefined => values.where((value) => value != $unknown).toList();
}
