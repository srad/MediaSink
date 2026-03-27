// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_filter_description_bool.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesFilterDescriptionBool _$ResponsesFilterDescriptionBoolFromJson(
  Map<String, dynamic> json,
) => ResponsesFilterDescriptionBool(
  description: json['description'] as String?,
  maxValue: json['maxValue'] as bool?,
  minValue: json['minValue'] as bool?,
  name: json['name'] as String?,
  range: json['range'] as String?,
  recommended: json['recommended'] as bool?,
);

Map<String, dynamic> _$ResponsesFilterDescriptionBoolToJson(
  ResponsesFilterDescriptionBool instance,
) => <String, dynamic>{
  'description': instance.description,
  'maxValue': instance.maxValue,
  'minValue': instance.minValue,
  'name': instance.name,
  'range': instance.range,
  'recommended': instance.recommended,
};
