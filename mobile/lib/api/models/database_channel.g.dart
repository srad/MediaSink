// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'database_channel.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DatabaseChannel _$DatabaseChannelFromJson(Map<String, dynamic> json) =>
    DatabaseChannel(
      channelId: (json['channelId'] as num?)?.toInt(),
      channelName: json['channelName'] as String?,
      createdAt: json['createdAt'] as String?,
      deleted: json['deleted'] as bool?,
      displayName: json['displayName'] as String?,
      fav: json['fav'] as bool?,
      isPaused: json['isPaused'] as bool?,
      minDuration: (json['minDuration'] as num?)?.toInt(),
      recordings:
          (json['recordings'] as List<dynamic>?)
              ?.map(
                (e) => DatabaseRecording.fromJson(e as Map<String, dynamic>),
              )
              .toList(),
      recordingsCount: (json['recordingsCount'] as num?)?.toInt(),
      recordingsSize: (json['recordingsSize'] as num?)?.toInt(),
      skipStart: (json['skipStart'] as num?)?.toInt(),
      tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList(),
      url: json['url'] as String?,
    );

Map<String, dynamic> _$DatabaseChannelToJson(DatabaseChannel instance) =>
    <String, dynamic>{
      'channelId': instance.channelId,
      'channelName': instance.channelName,
      'createdAt': instance.createdAt,
      'deleted': instance.deleted,
      'displayName': instance.displayName,
      'fav': instance.fav,
      'isPaused': instance.isPaused,
      'minDuration': instance.minDuration,
      'recordings': instance.recordings,
      'recordingsCount': instance.recordingsCount,
      'recordingsSize': instance.recordingsSize,
      'skipStart': instance.skipStart,
      'tags': instance.tags,
      'url': instance.url,
    };
