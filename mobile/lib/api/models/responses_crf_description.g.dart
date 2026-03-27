// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_crf_description.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesCrfDescription _$ResponsesCrfDescriptionFromJson(
  Map<String, dynamic> json,
) => ResponsesCrfDescription(
  approxRatio: json['approxRatio'] as num?,
  description: json['description'] as String?,
  label: json['label'] as String?,
  quality: json['quality'] as String?,
  value: (json['value'] as num?)?.toInt(),
);

Map<String, dynamic> _$ResponsesCrfDescriptionToJson(
  ResponsesCrfDescription instance,
) => <String, dynamic>{
  'approxRatio': instance.approxRatio,
  'description': instance.description,
  'label': instance.label,
  'quality': instance.quality,
  'value': instance.value,
};
