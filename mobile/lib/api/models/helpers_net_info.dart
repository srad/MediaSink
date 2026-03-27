// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

part 'helpers_net_info.g.dart';

@JsonSerializable()
class HelpersNetInfo {
  const HelpersNetInfo({
    this.createdAt,
    this.dev,
    this.receiveBytes,
    this.transmitBytes,
  });
  
  factory HelpersNetInfo.fromJson(Map<String, Object?> json) => _$HelpersNetInfoFromJson(json);
  
  final String? createdAt;
  final String? dev;
  final int? receiveBytes;
  final int? transmitBytes;

  Map<String, Object?> toJson() => _$HelpersNetInfoToJson(this);
}
