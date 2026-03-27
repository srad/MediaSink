// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'services_process_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ServicesProcessInfo _$ServicesProcessInfoFromJson(Map<String, dynamic> json) =>
    ServicesProcessInfo(
      args: json['args'] as String?,
      id: (json['id'] as num?)?.toInt(),
      output: json['output'] as String?,
      path: json['path'] as String?,
      pid: (json['pid'] as num?)?.toInt(),
    );

Map<String, dynamic> _$ServicesProcessInfoToJson(
  ServicesProcessInfo instance,
) => <String, dynamic>{
  'args': instance.args,
  'id': instance.id,
  'output': instance.output,
  'path': instance.path,
  'pid': instance.pid,
};
