// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_enhance_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsEnhanceRequest _$RequestsEnhanceRequestFromJson(
  Map<String, dynamic> json,
) => RequestsEnhanceRequest(
  denoiseStrength: json['denoiseStrength'] as num,
  encodingPreset: RequestsEnhanceRequestEncodingPreset.fromJson(
    json['encodingPreset'] as String,
  ),
  recordingId: (json['recordingId'] as num).toInt(),
  sharpenStrength: json['sharpenStrength'] as num,
  targetResolution: RequestsEnhanceRequestTargetResolution.fromJson(
    json['targetResolution'] as String,
  ),
  applyNormalize: json['applyNormalize'] as bool?,
  crf: (json['crf'] as num?)?.toInt(),
);

Map<String, dynamic> _$RequestsEnhanceRequestToJson(
  RequestsEnhanceRequest instance,
) => <String, dynamic>{
  'applyNormalize': instance.applyNormalize,
  'crf': instance.crf,
  'denoiseStrength': instance.denoiseStrength,
  'encodingPreset':
      _$RequestsEnhanceRequestEncodingPresetEnumMap[instance.encodingPreset]!,
  'recordingId': instance.recordingId,
  'sharpenStrength': instance.sharpenStrength,
  'targetResolution':
      _$RequestsEnhanceRequestTargetResolutionEnumMap[instance
          .targetResolution]!,
};

const _$RequestsEnhanceRequestEncodingPresetEnumMap = {
  RequestsEnhanceRequestEncodingPreset.veryfast: 'veryfast',
  RequestsEnhanceRequestEncodingPreset.faster: 'faster',
  RequestsEnhanceRequestEncodingPreset.fast: 'fast',
  RequestsEnhanceRequestEncodingPreset.medium: 'medium',
  RequestsEnhanceRequestEncodingPreset.slow: 'slow',
  RequestsEnhanceRequestEncodingPreset.slower: 'slower',
  RequestsEnhanceRequestEncodingPreset.veryslow: 'veryslow',
  RequestsEnhanceRequestEncodingPreset.$unknown: r'$unknown',
};

const _$RequestsEnhanceRequestTargetResolutionEnumMap = {
  RequestsEnhanceRequestTargetResolution.value720p: '720p',
  RequestsEnhanceRequestTargetResolution.value1080p: '1080p',
  RequestsEnhanceRequestTargetResolution.value1440p: '1440p',
  RequestsEnhanceRequestTargetResolution.value4k: '4k',
  RequestsEnhanceRequestTargetResolution.$unknown: r'$unknown',
};
