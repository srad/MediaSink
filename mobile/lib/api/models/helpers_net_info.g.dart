// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'helpers_net_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

HelpersNetInfo _$HelpersNetInfoFromJson(Map<String, dynamic> json) =>
    HelpersNetInfo(
      createdAt: json['createdAt'] as String?,
      dev: json['dev'] as String?,
      receiveBytes: (json['receiveBytes'] as num?)?.toInt(),
      transmitBytes: (json['transmitBytes'] as num?)?.toInt(),
    );

Map<String, dynamic> _$HelpersNetInfoToJson(HelpersNetInfo instance) =>
    <String, dynamic>{
      'createdAt': instance.createdAt,
      'dev': instance.dev,
      'receiveBytes': instance.receiveBytes,
      'transmitBytes': instance.transmitBytes,
    };
