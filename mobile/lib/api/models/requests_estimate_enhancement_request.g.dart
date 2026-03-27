// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_estimate_enhancement_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsEstimateEnhancementRequest _$RequestsEstimateEnhancementRequestFromJson(
  Map<String, dynamic> json,
) => RequestsEstimateEnhancementRequest(
  denoiseStrength: json['denoiseStrength'] as num,
  encodingPreset: RequestsEstimateEnhancementRequestEncodingPreset.fromJson(
    json['encodingPreset'] as String,
  ),
  sharpenStrength: json['sharpenStrength'] as num,
  targetResolution: RequestsEstimateEnhancementRequestTargetResolution.fromJson(
    json['targetResolution'] as String,
  ),
  applyNormalize: json['applyNormalize'] as bool?,
  crf: (json['crf'] as num?)?.toInt(),
);

Map<String, dynamic> _$RequestsEstimateEnhancementRequestToJson(
  RequestsEstimateEnhancementRequest instance,
) => <String, dynamic>{
  'applyNormalize': instance.applyNormalize,
  'crf': instance.crf,
  'denoiseStrength': instance.denoiseStrength,
  'encodingPreset':
      _$RequestsEstimateEnhancementRequestEncodingPresetEnumMap[instance
          .encodingPreset]!,
  'sharpenStrength': instance.sharpenStrength,
  'targetResolution':
      _$RequestsEstimateEnhancementRequestTargetResolutionEnumMap[instance
          .targetResolution]!,
};

const _$RequestsEstimateEnhancementRequestEncodingPresetEnumMap = {
  RequestsEstimateEnhancementRequestEncodingPreset.veryfast: 'veryfast',
  RequestsEstimateEnhancementRequestEncodingPreset.faster: 'faster',
  RequestsEstimateEnhancementRequestEncodingPreset.fast: 'fast',
  RequestsEstimateEnhancementRequestEncodingPreset.medium: 'medium',
  RequestsEstimateEnhancementRequestEncodingPreset.slow: 'slow',
  RequestsEstimateEnhancementRequestEncodingPreset.slower: 'slower',
  RequestsEstimateEnhancementRequestEncodingPreset.veryslow: 'veryslow',
  RequestsEstimateEnhancementRequestEncodingPreset.$unknown: r'$unknown',
};

const _$RequestsEstimateEnhancementRequestTargetResolutionEnumMap = {
  RequestsEstimateEnhancementRequestTargetResolution.value720p: '720p',
  RequestsEstimateEnhancementRequestTargetResolution.value1080p: '1080p',
  RequestsEstimateEnhancementRequestTargetResolution.value1440p: '1440p',
  RequestsEstimateEnhancementRequestTargetResolution.value4k: '4k',
  RequestsEstimateEnhancementRequestTargetResolution.$unknown: r'$unknown',
};
