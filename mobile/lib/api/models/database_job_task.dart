// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

@JsonEnum()
enum DatabaseJobTask {
  @JsonValue('convert')
  convert('convert'),
  @JsonValue('preview-cover')
  previewCover('preview-cover'),
  @JsonValue('preview-stripe')
  previewStripe('preview-stripe'),
  @JsonValue('preview-video')
  previewVideo('preview-video'),
  @JsonValue('cut')
  cut('cut'),
  /// Default value for all unparsed values, allows backward compatibility when adding new values on the backend.
  $unknown(null);

  const DatabaseJobTask(this.json);

  factory DatabaseJobTask.fromJson(String json) => values.firstWhere(
        (e) => e.json == json,
        orElse: () => $unknown,
      );

  final String? json;
}
