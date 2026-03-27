// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_filter_descriptions.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesFilterDescriptions _$ResponsesFilterDescriptionsFromJson(
  Map<String, dynamic> json,
) => ResponsesFilterDescriptions(
  applyNormalize:
      json['applyNormalize'] == null
          ? null
          : ResponsesFilterDescriptionBool.fromJson(
            json['applyNormalize'] as Map<String, dynamic>,
          ),
  denoiseStrength:
      json['denoiseStrength'] == null
          ? null
          : ResponsesFilterDescriptionFloat.fromJson(
            json['denoiseStrength'] as Map<String, dynamic>,
          ),
  sharpenStrength:
      json['sharpenStrength'] == null
          ? null
          : ResponsesFilterDescriptionFloat.fromJson(
            json['sharpenStrength'] as Map<String, dynamic>,
          ),
);

Map<String, dynamic> _$ResponsesFilterDescriptionsToJson(
  ResponsesFilterDescriptions instance,
) => <String, dynamic>{
  'applyNormalize': instance.applyNormalize,
  'denoiseStrength': instance.denoiseStrength,
  'sharpenStrength': instance.sharpenStrength,
};
