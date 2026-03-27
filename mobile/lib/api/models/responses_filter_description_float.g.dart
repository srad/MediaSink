// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_filter_description_float.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesFilterDescriptionFloat _$ResponsesFilterDescriptionFloatFromJson(
  Map<String, dynamic> json,
) => ResponsesFilterDescriptionFloat(
  description: json['description'] as String?,
  maxValue: json['maxValue'] as num?,
  minValue: json['minValue'] as num?,
  name: json['name'] as String?,
  range: json['range'] as String?,
  recommended: json['recommended'] as num?,
);

Map<String, dynamic> _$ResponsesFilterDescriptionFloatToJson(
  ResponsesFilterDescriptionFloat instance,
) => <String, dynamic>{
  'description': instance.description,
  'maxValue': instance.maxValue,
  'minValue': instance.minValue,
  'name': instance.name,
  'range': instance.range,
  'recommended': instance.recommended,
};
