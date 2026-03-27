// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'util_net_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

UtilNetInfo _$UtilNetInfoFromJson(Map<String, dynamic> json) => UtilNetInfo(
  createdAt: json['createdAt'] as String?,
  dev: json['dev'] as String?,
  receiveBytes: (json['receiveBytes'] as num?)?.toInt(),
  transmitBytes: (json['transmitBytes'] as num?)?.toInt(),
);

Map<String, dynamic> _$UtilNetInfoToJson(UtilNetInfo instance) =>
    <String, dynamic>{
      'createdAt': instance.createdAt,
      'dev': instance.dev,
      'receiveBytes': instance.receiveBytes,
      'transmitBytes': instance.transmitBytes,
    };
