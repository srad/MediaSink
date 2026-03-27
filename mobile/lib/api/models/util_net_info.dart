// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'util_net_info.g.dart';

@JsonSerializable()
class UtilNetInfo {
  const UtilNetInfo({
    this.createdAt,
    this.dev,
    this.receiveBytes,
    this.transmitBytes,
  });
  
  factory UtilNetInfo.fromJson(Map<String, Object?> json) => _$UtilNetInfoFromJson(json);
  
  final String? createdAt;
  final String? dev;
  final int? receiveBytes;
  final int? transmitBytes;

  Map<String, Object?> toJson() => _$UtilNetInfoToJson(this);
}
