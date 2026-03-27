// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'util_cpu_load.g.dart';

@JsonSerializable()
class UtilCpuLoad {
  const UtilCpuLoad({
    this.cpu,
    this.createdAt,
    this.load,
  });
  
  factory UtilCpuLoad.fromJson(Map<String, Object?> json) => _$UtilCpuLoadFromJson(json);
  
  final String? cpu;
  final String? createdAt;
  final num? load;

  Map<String, Object?> toJson() => _$UtilCpuLoadToJson(this);
}
