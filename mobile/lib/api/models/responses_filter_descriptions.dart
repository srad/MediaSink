// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'responses_filter_description_bool.dart';
import 'responses_filter_description_float.dart';

part 'responses_filter_descriptions.g.dart';

@JsonSerializable()
class ResponsesFilterDescriptions {
  const ResponsesFilterDescriptions({
    this.applyNormalize,
    this.denoiseStrength,
    this.sharpenStrength,
  });
  
  factory ResponsesFilterDescriptions.fromJson(Map<String, Object?> json) => _$ResponsesFilterDescriptionsFromJson(json);
  
  final ResponsesFilterDescriptionBool? applyNormalize;
  final ResponsesFilterDescriptionFloat? denoiseStrength;
  final ResponsesFilterDescriptionFloat? sharpenStrength;

  Map<String, Object?> toJson() => _$ResponsesFilterDescriptionsToJson(this);
}
