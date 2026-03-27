// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_resolution_description.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesResolutionDescription _$ResponsesResolutionDescriptionFromJson(
  Map<String, dynamic> json,
) => ResponsesResolutionDescription(
  description: json['description'] as String?,
  dimensions: json['dimensions'] as String?,
  resolution: json['resolution'] as String?,
  useCase: json['useCase'] as String?,
);

Map<String, dynamic> _$ResponsesResolutionDescriptionToJson(
  ResponsesResolutionDescription instance,
) => <String, dynamic>{
  'description': instance.description,
  'dimensions': instance.dimensions,
  'resolution': instance.resolution,
  'useCase': instance.useCase,
};
