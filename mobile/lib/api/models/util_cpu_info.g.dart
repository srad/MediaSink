// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'util_cpu_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

UtilCpuInfo _$UtilCpuInfoFromJson(Map<String, dynamic> json) => UtilCpuInfo(
  loadCpu:
      (json['loadCpu'] as List<dynamic>?)
          ?.map((e) => UtilCpuLoad.fromJson(e as Map<String, dynamic>))
          .toList(),
);

Map<String, dynamic> _$UtilCpuInfoToJson(UtilCpuInfo instance) =>
    <String, dynamic>{'loadCpu': instance.loadCpu};
