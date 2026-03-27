// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'util_cpu_load.dart';

part 'util_cpu_info.g.dart';

@JsonSerializable()
class UtilCpuInfo {
  const UtilCpuInfo({
    this.loadCpu,
  });
  
  factory UtilCpuInfo.fromJson(Map<String, Object?> json) => _$UtilCpuInfoFromJson(json);
  
  final List<UtilCpuLoad>? loadCpu;

  Map<String, Object?> toJson() => _$UtilCpuInfoToJson(this);
}
