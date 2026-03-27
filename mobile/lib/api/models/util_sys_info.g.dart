// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'util_sys_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

UtilSysInfo _$UtilSysInfoFromJson(Map<String, dynamic> json) => UtilSysInfo(
  cpuInfo:
      json['cpuInfo'] == null
          ? null
          : UtilCpuInfo.fromJson(json['cpuInfo'] as Map<String, dynamic>),
  diskInfo:
      json['diskInfo'] == null
          ? null
          : UtilDiskInfo.fromJson(json['diskInfo'] as Map<String, dynamic>),
  netInfo:
      json['netInfo'] == null
          ? null
          : UtilNetInfo.fromJson(json['netInfo'] as Map<String, dynamic>),
);

Map<String, dynamic> _$UtilSysInfoToJson(UtilSysInfo instance) =>
    <String, dynamic>{
      'cpuInfo': instance.cpuInfo,
      'diskInfo': instance.diskInfo,
      'netInfo': instance.netInfo,
    };
