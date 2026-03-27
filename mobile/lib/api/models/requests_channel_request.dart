// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'requests_channel_request.g.dart';

@JsonSerializable()
class RequestsChannelRequest {
  const RequestsChannelRequest({
    this.channelName,
    this.deleted,
    this.displayName,
    this.fav,
    this.isPaused,
    this.minDuration,
    this.skipStart,
    this.tags,
    this.url,
  });
  
  factory RequestsChannelRequest.fromJson(Map<String, Object?> json) => _$RequestsChannelRequestFromJson(json);
  
  final String? channelName;
  final bool? deleted;
  final String? displayName;
  final bool? fav;
  final bool? isPaused;
  final int? minDuration;
  final int? skipStart;
  final List<String>? tags;
  final String? url;

  Map<String, Object?> toJson() => _$RequestsChannelRequestToJson(this);
}
