// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_filter_description_float.g.dart';

@JsonSerializable()
class ResponsesFilterDescriptionFloat {
  const ResponsesFilterDescriptionFloat({
    this.description,
    this.maxValue,
    this.minValue,
    this.name,
    this.range,
    this.recommended,
  });
  
  factory ResponsesFilterDescriptionFloat.fromJson(Map<String, Object?> json) => _$ResponsesFilterDescriptionFloatFromJson(json);
  
  final String? description;
  final num? maxValue;
  final num? minValue;
  final String? name;
  final String? range;
  final num? recommended;

  Map<String, Object?> toJson() => _$ResponsesFilterDescriptionFloatToJson(this);
}
