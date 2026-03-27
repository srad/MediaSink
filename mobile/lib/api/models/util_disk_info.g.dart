// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'util_disk_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

UtilDiskInfo _$UtilDiskInfoFromJson(Map<String, dynamic> json) => UtilDiskInfo(
  availFormattedGb: (json['availFormattedGb'] as num?)?.toInt(),
  pcent: (json['pcent'] as num?)?.toInt(),
  sizeFormattedGb: (json['sizeFormattedGb'] as num?)?.toInt(),
  usedFormattedGb: (json['usedFormattedGb'] as num?)?.toInt(),
);

Map<String, dynamic> _$UtilDiskInfoToJson(UtilDiskInfo instance) =>
    <String, dynamic>{
      'availFormattedGb': instance.availFormattedGb,
      'pcent': instance.pcent,
      'sizeFormattedGb': instance.sizeFormattedGb,
      'usedFormattedGb': instance.usedFormattedGb,
    };
