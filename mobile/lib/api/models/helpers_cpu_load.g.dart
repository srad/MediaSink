// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'helpers_cpu_load.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

HelpersCpuLoad _$HelpersCpuLoadFromJson(Map<String, dynamic> json) =>
    HelpersCpuLoad(
      cpu: json['cpu'] as String?,
      createdAt: json['createdAt'] as String?,
      load: json['load'] as num?,
    );

Map<String, dynamic> _$HelpersCpuLoadToJson(HelpersCpuLoad instance) =>
    <String, dynamic>{
      'cpu': instance.cpu,
      'createdAt': instance.createdAt,
      'load': instance.load,
    };
