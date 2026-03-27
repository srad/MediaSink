// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

part 'helpers_cpu_load.g.dart';

@JsonSerializable()
class HelpersCpuLoad {
  const HelpersCpuLoad({
    this.cpu,
    this.createdAt,
    this.load,
  });
  
  factory HelpersCpuLoad.fromJson(Map<String, Object?> json) => _$HelpersCpuLoadFromJson(json);
  
  final String? cpu;
  final String? createdAt;
  final num? load;

  Map<String, Object?> toJson() => _$HelpersCpuLoadToJson(this);
}
