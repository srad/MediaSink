// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_preset_description.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesPresetDescription _$ResponsesPresetDescriptionFromJson(
  Map<String, dynamic> json,
) => ResponsesPresetDescription(
  description: json['description'] as String?,
  encodeSpeed: json['encodeSpeed'] as String?,
  label: json['label'] as String?,
  preset: json['preset'] as String?,
);

Map<String, dynamic> _$ResponsesPresetDescriptionToJson(
  ResponsesPresetDescription instance,
) => <String, dynamic>{
  'description': instance.description,
  'encodeSpeed': instance.encodeSpeed,
  'label': instance.label,
  'preset': instance.preset,
};
