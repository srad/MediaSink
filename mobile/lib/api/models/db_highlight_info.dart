// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'db_highlight_info.g.dart';

@JsonSerializable()
class DbHighlightInfo {
  const DbHighlightInfo({
    this.endTime,
    this.intensity,
    this.startTime,
    this.timestamp,
    this.type,
  });
  
  factory DbHighlightInfo.fromJson(Map<String, Object?> json) => _$DbHighlightInfoFromJson(json);
  
  final num? endTime;

  /// 0-1, higher = more activity
  final num? intensity;
  final num? startTime;
  final num? timestamp;

  /// "motion", "sceneChange", "transition"
  final String? type;

  Map<String, Object?> toJson() => _$DbHighlightInfoToJson(this);
}
