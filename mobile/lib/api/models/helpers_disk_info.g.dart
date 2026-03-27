// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'helpers_disk_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

HelpersDiskInfo _$HelpersDiskInfoFromJson(Map<String, dynamic> json) =>
    HelpersDiskInfo(
      availFormattedGb: (json['availFormattedGb'] as num?)?.toInt(),
      pcent: (json['pcent'] as num?)?.toInt(),
      sizeFormattedGb: (json['sizeFormattedGb'] as num?)?.toInt(),
      usedFormattedGb: (json['usedFormattedGb'] as num?)?.toInt(),
    );

Map<String, dynamic> _$HelpersDiskInfoToJson(HelpersDiskInfo instance) =>
    <String, dynamic>{
      'availFormattedGb': instance.availFormattedGb,
      'pcent': instance.pcent,
      'sizeFormattedGb': instance.sizeFormattedGb,
      'usedFormattedGb': instance.usedFormattedGb,
    };
