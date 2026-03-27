// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'db_segment_info.g.dart';

@JsonSerializable()
class DbSegmentInfo {
  const DbSegmentInfo({
    this.confidence,
    this.endTime,
    this.kind,
    this.representativeTimestamp,
    this.startTime,
  });
  
  factory DbSegmentInfo.fromJson(Map<String, Object?> json) => _$DbSegmentInfoFromJson(json);
  
  /// 0-1, higher = stronger boundary evidence
  final num? confidence;
  final num? endTime;
  final String? kind;
  final num? representativeTimestamp;
  final num? startTime;

  Map<String, Object?> toJson() => _$DbSegmentInfoToJson(this);
}
