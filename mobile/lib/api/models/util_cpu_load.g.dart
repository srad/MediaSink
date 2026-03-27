// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'util_cpu_load.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

UtilCpuLoad _$UtilCpuLoadFromJson(Map<String, dynamic> json) => UtilCpuLoad(
  cpu: json['cpu'] as String?,
  createdAt: json['createdAt'] as String?,
  load: json['load'] as num?,
);

Map<String, dynamic> _$UtilCpuLoadToJson(UtilCpuLoad instance) =>
    <String, dynamic>{
      'cpu': instance.cpu,
      'createdAt': instance.createdAt,
      'load': instance.load,
    };
