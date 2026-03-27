// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'db_scene_info.g.dart';

@JsonSerializable()
class DbSceneInfo {
  const DbSceneInfo({
    this.changeIntensity,
    this.endTime,
    this.startTime,
  });
  
  factory DbSceneInfo.fromJson(Map<String, Object?> json) => _$DbSceneInfoFromJson(json);
  
  /// 0-1, higher = more change
  final num? changeIntensity;
  final num? endTime;
  final num? startTime;

  Map<String, Object?> toJson() => _$DbSceneInfoToJson(this);
}
