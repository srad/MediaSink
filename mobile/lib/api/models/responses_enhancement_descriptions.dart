// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'responses_crf_description.dart';
import 'responses_filter_descriptions.dart';
import 'responses_preset_description.dart';
import 'responses_resolution_description.dart';

part 'responses_enhancement_descriptions.g.dart';

@JsonSerializable()
class ResponsesEnhancementDescriptions {
  const ResponsesEnhancementDescriptions({
    this.crfValues,
    this.filters,
    this.presets,
    this.resolutions,
  });
  
  factory ResponsesEnhancementDescriptions.fromJson(Map<String, Object?> json) => _$ResponsesEnhancementDescriptionsFromJson(json);
  
  final List<ResponsesCrfDescription>? crfValues;
  final ResponsesFilterDescriptions? filters;
  final List<ResponsesPresetDescription>? presets;
  final List<ResponsesResolutionDescription>? resolutions;

  Map<String, Object?> toJson() => _$ResponsesEnhancementDescriptionsToJson(this);
}
