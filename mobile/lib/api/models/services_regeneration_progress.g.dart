// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'services_regeneration_progress.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ServicesRegenerationProgress _$ServicesRegenerationProgressFromJson(
  Map<String, dynamic> json,
) => ServicesRegenerationProgress(
  current: (json['current'] as num?)?.toInt(),
  currentVideo: json['currentVideo'] as String?,
  isRunning: json['isRunning'] as bool?,
  total: (json['total'] as num?)?.toInt(),
);

Map<String, dynamic> _$ServicesRegenerationProgressToJson(
  ServicesRegenerationProgress instance,
) => <String, dynamic>{
  'current': instance.current,
  'currentVideo': instance.currentVideo,
  'isRunning': instance.isRunning,
  'total': instance.total,
};
