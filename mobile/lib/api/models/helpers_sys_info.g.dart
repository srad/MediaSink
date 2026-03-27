// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'helpers_sys_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

HelpersSysInfo _$HelpersSysInfoFromJson(
  Map<String, dynamic> json,
) => HelpersSysInfo(
  cpuInfo:
      json['cpuInfo'] == null
          ? null
          : HelpersCpuInfo.fromJson(json['cpuInfo'] as Map<String, dynamic>),
  diskInfo:
      json['diskInfo'] == null
          ? null
          : HelpersDiskInfo.fromJson(json['diskInfo'] as Map<String, dynamic>),
  netInfo:
      json['netInfo'] == null
          ? null
          : HelpersNetInfo.fromJson(json['netInfo'] as Map<String, dynamic>),
);

Map<String, dynamic> _$HelpersSysInfoToJson(HelpersSysInfo instance) =>
    <String, dynamic>{
      'cpuInfo': instance.cpuInfo,
      'diskInfo': instance.diskInfo,
      'netInfo': instance.netInfo,
    };
