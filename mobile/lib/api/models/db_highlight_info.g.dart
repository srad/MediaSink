// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'db_highlight_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DbHighlightInfo _$DbHighlightInfoFromJson(Map<String, dynamic> json) =>
    DbHighlightInfo(
      endTime: json['endTime'] as num?,
      intensity: json['intensity'] as num?,
      startTime: json['startTime'] as num?,
      timestamp: json['timestamp'] as num?,
      type: json['type'] as String?,
    );

Map<String, dynamic> _$DbHighlightInfoToJson(DbHighlightInfo instance) =>
    <String, dynamic>{
      'endTime': instance.endTime,
      'intensity': instance.intensity,
      'startTime': instance.startTime,
      'timestamp': instance.timestamp,
      'type': instance.type,
    };
