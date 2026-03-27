// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'db_scene_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DbSceneInfo _$DbSceneInfoFromJson(Map<String, dynamic> json) => DbSceneInfo(
  changeIntensity: json['changeIntensity'] as num?,
  endTime: json['endTime'] as num?,
  startTime: json['startTime'] as num?,
);

Map<String, dynamic> _$DbSceneInfoToJson(DbSceneInfo instance) =>
    <String, dynamic>{
      'changeIntensity': instance.changeIntensity,
      'endTime': instance.endTime,
      'startTime': instance.startTime,
    };
