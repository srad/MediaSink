// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_channel_tags_update_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsChannelTagsUpdateRequest _$RequestsChannelTagsUpdateRequestFromJson(
  Map<String, dynamic> json,
) => RequestsChannelTagsUpdateRequest(
  tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList(),
);

Map<String, dynamic> _$RequestsChannelTagsUpdateRequestToJson(
  RequestsChannelTagsUpdateRequest instance,
) => <String, dynamic>{'tags': instance.tags};
