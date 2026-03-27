// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

import 'helpers_cpu_info.dart';
import 'helpers_disk_info.dart';
import 'helpers_net_info.dart';

part 'helpers_sys_info.g.dart';

@JsonSerializable()
class HelpersSysInfo {
  const HelpersSysInfo({
    this.cpuInfo,
    this.diskInfo,
    this.netInfo,
  });
  
  factory HelpersSysInfo.fromJson(Map<String, Object?> json) => _$HelpersSysInfoFromJson(json);
  
  final HelpersCpuInfo? cpuInfo;
  final HelpersDiskInfo? diskInfo;
  final HelpersNetInfo? netInfo;

  Map<String, Object?> toJson() => _$HelpersSysInfoToJson(this);
}
