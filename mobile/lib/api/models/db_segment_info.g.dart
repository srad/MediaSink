// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'db_segment_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DbSegmentInfo _$DbSegmentInfoFromJson(Map<String, dynamic> json) =>
    DbSegmentInfo(
      confidence: json['confidence'] as num?,
      endTime: json['endTime'] as num?,
      kind: json['kind'] as String?,
      representativeTimestamp: json['representativeTimestamp'] as num?,
      startTime: json['startTime'] as num?,
    );

Map<String, dynamic> _$DbSegmentInfoToJson(DbSegmentInfo instance) =>
    <String, dynamic>{
      'confidence': instance.confidence,
      'endTime': instance.endTime,
      'kind': instance.kind,
      'representativeTimestamp': instance.representativeTimestamp,
      'startTime': instance.startTime,
    };
