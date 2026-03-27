// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'requests_enhance_request_encoding_preset.dart';
import 'requests_enhance_request_target_resolution.dart';

part 'requests_enhance_request.g.dart';

@JsonSerializable()
class RequestsEnhanceRequest {
  const RequestsEnhanceRequest({
    required this.denoiseStrength,
    required this.encodingPreset,
    required this.recordingId,
    required this.sharpenStrength,
    required this.targetResolution,
    this.applyNormalize,
    this.crf,
  });
  
  factory RequestsEnhanceRequest.fromJson(Map<String, Object?> json) => _$RequestsEnhanceRequestFromJson(json);
  
  final bool? applyNormalize;
  final int? crf;
  final num denoiseStrength;
  final RequestsEnhanceRequestEncodingPreset encodingPreset;
  final int recordingId;
  final num sharpenStrength;
  final RequestsEnhanceRequestTargetResolution targetResolution;

  Map<String, Object?> toJson() => _$RequestsEnhanceRequestToJson(this);
}
