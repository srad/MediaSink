// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'services_channel_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ServicesChannelInfo _$ServicesChannelInfoFromJson(Map<String, dynamic> json) =>
    ServicesChannelInfo(
      channelId: (json['channelId'] as num?)?.toInt(),
      channelName: json['channelName'] as String?,
      createdAt: json['createdAt'] as String?,
      deleted: json['deleted'] as bool?,
      displayName: json['displayName'] as String?,
      fav: json['fav'] as bool?,
      isOnline: json['isOnline'] as bool?,
      isPaused: json['isPaused'] as bool?,
      isRecording: json['isRecording'] as bool?,
      isTerminating: json['isTerminating'] as bool?,
      minDuration: (json['minDuration'] as num?)?.toInt(),
      minRecording: json['minRecording'] as num?,
      preview: json['preview'] as String?,
      recordings:
          (json['recordings'] as List<dynamic>?)
              ?.map((e) => DbRecording.fromJson(e as Map<String, dynamic>))
              .toList(),
      recordingsCount: (json['recordingsCount'] as num?)?.toInt(),
      recordingsSize: (json['recordingsSize'] as num?)?.toInt(),
      skipStart: (json['skipStart'] as num?)?.toInt(),
      tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList(),
      url: json['url'] as String?,
    );

Map<String, dynamic> _$ServicesChannelInfoToJson(
  ServicesChannelInfo instance,
) => <String, dynamic>{
  'channelId': instance.channelId,
  'channelName': instance.channelName,
  'createdAt': instance.createdAt,
  'deleted': instance.deleted,
  'displayName': instance.displayName,
  'fav': instance.fav,
  'isOnline': instance.isOnline,
  'isPaused': instance.isPaused,
  'isRecording': instance.isRecording,
  'isTerminating': instance.isTerminating,
  'minDuration': instance.minDuration,
  'minRecording': instance.minRecording,
  'preview': instance.preview,
  'recordings': instance.recordings,
  'recordingsCount': instance.recordingsCount,
  'recordingsSize': instance.recordingsSize,
  'skipStart': instance.skipStart,
  'tags': instance.tags,
  'url': instance.url,
};
