// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

part 'helpers_disk_info.g.dart';

@JsonSerializable()
class HelpersDiskInfo {
  const HelpersDiskInfo({
    this.availFormattedGb,
    this.pcent,
    this.sizeFormattedGb,
    this.usedFormattedGb,
  });
  
  factory HelpersDiskInfo.fromJson(Map<String, Object?> json) => _$HelpersDiskInfoFromJson(json);
  
  final int? availFormattedGb;
  final int? pcent;
  final int? sizeFormattedGb;
  final int? usedFormattedGb;

  Map<String, Object?> toJson() => _$HelpersDiskInfoToJson(this);
}
