// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_preset_description.g.dart';

@JsonSerializable()
class ResponsesPresetDescription {
  const ResponsesPresetDescription({
    this.description,
    this.encodeSpeed,
    this.label,
    this.preset,
  });
  
  factory ResponsesPresetDescription.fromJson(Map<String, Object?> json) => _$ResponsesPresetDescriptionFromJson(json);
  
  final String? description;
  final String? encodeSpeed;
  final String? label;
  final String? preset;

  Map<String, Object?> toJson() => _$ResponsesPresetDescriptionToJson(this);
}
