// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_enhancement_descriptions.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesEnhancementDescriptions _$ResponsesEnhancementDescriptionsFromJson(
  Map<String, dynamic> json,
) => ResponsesEnhancementDescriptions(
  crfValues:
      (json['crfValues'] as List<dynamic>?)
          ?.map(
            (e) => ResponsesCrfDescription.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
  filters:
      json['filters'] == null
          ? null
          : ResponsesFilterDescriptions.fromJson(
            json['filters'] as Map<String, dynamic>,
          ),
  presets:
      (json['presets'] as List<dynamic>?)
          ?.map(
            (e) =>
                ResponsesPresetDescription.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
  resolutions:
      (json['resolutions'] as List<dynamic>?)
          ?.map(
            (e) => ResponsesResolutionDescription.fromJson(
              e as Map<String, dynamic>,
            ),
          )
          .toList(),
);

Map<String, dynamic> _$ResponsesEnhancementDescriptionsToJson(
  ResponsesEnhancementDescriptions instance,
) => <String, dynamic>{
  'crfValues': instance.crfValues,
  'filters': instance.filters,
  'presets': instance.presets,
  'resolutions': instance.resolutions,
};
