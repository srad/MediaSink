// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_channel_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsChannelRequest _$RequestsChannelRequestFromJson(
  Map<String, dynamic> json,
) => RequestsChannelRequest(
  channelName: json['channelName'] as String?,
  deleted: json['deleted'] as bool?,
  displayName: json['displayName'] as String?,
  fav: json['fav'] as bool?,
  isPaused: json['isPaused'] as bool?,
  minDuration: (json['minDuration'] as num?)?.toInt(),
  skipStart: (json['skipStart'] as num?)?.toInt(),
  tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList(),
  url: json['url'] as String?,
);

Map<String, dynamic> _$RequestsChannelRequestToJson(
  RequestsChannelRequest instance,
) => <String, dynamic>{
  'channelName': instance.channelName,
  'deleted': instance.deleted,
  'displayName': instance.displayName,
  'fav': instance.fav,
  'isPaused': instance.isPaused,
  'minDuration': instance.minDuration,
  'skipStart': instance.skipStart,
  'tags': instance.tags,
  'url': instance.url,
};
