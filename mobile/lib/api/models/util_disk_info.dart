// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'util_disk_info.g.dart';

@JsonSerializable()
class UtilDiskInfo {
  const UtilDiskInfo({
    this.availFormattedGb,
    this.pcent,
    this.sizeFormattedGb,
    this.usedFormattedGb,
  });
  
  factory UtilDiskInfo.fromJson(Map<String, Object?> json) => _$UtilDiskInfoFromJson(json);
  
  final int? availFormattedGb;
  final int? pcent;
  final int? sizeFormattedGb;
  final int? usedFormattedGb;

  Map<String, Object?> toJson() => _$UtilDiskInfoToJson(this);
}
