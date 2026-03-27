// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_filter_description_bool.g.dart';

@JsonSerializable()
class ResponsesFilterDescriptionBool {
  const ResponsesFilterDescriptionBool({
    this.description,
    this.maxValue,
    this.minValue,
    this.name,
    this.range,
    this.recommended,
  });
  
  factory ResponsesFilterDescriptionBool.fromJson(Map<String, Object?> json) => _$ResponsesFilterDescriptionBoolFromJson(json);
  
  final String? description;
  final bool? maxValue;
  final bool? minValue;
  final String? name;
  final String? range;
  final bool? recommended;

  Map<String, Object?> toJson() => _$ResponsesFilterDescriptionBoolToJson(this);
}
