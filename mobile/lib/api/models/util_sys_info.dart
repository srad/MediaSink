// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'util_cpu_info.dart';
import 'util_disk_info.dart';
import 'util_net_info.dart';

part 'util_sys_info.g.dart';

@JsonSerializable()
class UtilSysInfo {
  const UtilSysInfo({
    this.cpuInfo,
    this.diskInfo,
    this.netInfo,
  });
  
  factory UtilSysInfo.fromJson(Map<String, Object?> json) => _$UtilSysInfoFromJson(json);
  
  final UtilCpuInfo? cpuInfo;
  final UtilDiskInfo? diskInfo;
  final UtilNetInfo? netInfo;

  Map<String, Object?> toJson() => _$UtilSysInfoToJson(this);
}
