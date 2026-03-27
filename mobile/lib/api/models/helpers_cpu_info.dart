// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

import 'helpers_cpu_load.dart';

part 'helpers_cpu_info.g.dart';

@JsonSerializable()
class HelpersCpuInfo {
  const HelpersCpuInfo({
    this.loadCpu,
  });
  
  factory HelpersCpuInfo.fromJson(Map<String, Object?> json) => _$HelpersCpuInfoFromJson(json);
  
  final List<HelpersCpuLoad>? loadCpu;

  Map<String, Object?> toJson() => _$HelpersCpuInfoToJson(this);
}
