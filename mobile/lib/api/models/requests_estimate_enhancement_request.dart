// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'requests_estimate_enhancement_request_encoding_preset.dart';
import 'requests_estimate_enhancement_request_target_resolution.dart';

part 'requests_estimate_enhancement_request.g.dart';

@JsonSerializable()
class RequestsEstimateEnhancementRequest {
  const RequestsEstimateEnhancementRequest({
    required this.denoiseStrength,
    required this.encodingPreset,
    required this.sharpenStrength,
    required this.targetResolution,
    this.applyNormalize,
    this.crf,
  });
  
  factory RequestsEstimateEnhancementRequest.fromJson(Map<String, Object?> json) => _$RequestsEstimateEnhancementRequestFromJson(json);
  
  final bool? applyNormalize;
  final int? crf;
  final num denoiseStrength;
  final RequestsEstimateEnhancementRequestEncodingPreset encodingPreset;
  final num sharpenStrength;
  final RequestsEstimateEnhancementRequestTargetResolution targetResolution;

  Map<String, Object?> toJson() => _$RequestsEstimateEnhancementRequestToJson(this);
}
