// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'helpers_cpu_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

HelpersCpuInfo _$HelpersCpuInfoFromJson(Map<String, dynamic> json) =>
    HelpersCpuInfo(
      loadCpu:
          (json['loadCpu'] as List<dynamic>?)
              ?.map((e) => HelpersCpuLoad.fromJson(e as Map<String, dynamic>))
              .toList(),
    );

Map<String, dynamic> _$HelpersCpuInfoToJson(HelpersCpuInfo instance) =>
    <String, dynamic>{'loadCpu': instance.loadCpu};
