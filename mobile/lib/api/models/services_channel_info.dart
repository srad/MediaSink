// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_recording.dart';

part 'services_channel_info.g.dart';

@JsonSerializable()
class ServicesChannelInfo {
  const ServicesChannelInfo({
    this.channelId,
    this.channelName,
    this.createdAt,
    this.deleted,
    this.displayName,
    this.fav,
    this.isOnline,
    this.isPaused,
    this.isRecording,
    this.isTerminating,
    this.minDuration,
    this.minRecording,
    this.preview,
    this.recordings,
    this.recordingsCount,
    this.recordingsSize,
    this.skipStart,
    this.tags,
    this.url,
  });
  
  factory ServicesChannelInfo.fromJson(Map<String, Object?> json) => _$ServicesChannelInfoFromJson(json);
  
  final int? channelId;
  final String? channelName;
  final String? createdAt;
  final bool? deleted;
  final String? displayName;
  final bool? fav;
  final bool? isOnline;
  final bool? isPaused;
  final bool? isRecording;
  final bool? isTerminating;
  final int? minDuration;
  final num? minRecording;
  final String? preview;

  /// 1:n
  final List<DbRecording>? recordings;

  /// Only for query result.
  final int? recordingsCount;
  final int? recordingsSize;
  final int? skipStart;
  final List<String>? tags;
  final String? url;

  Map<String, Object?> toJson() => _$ServicesChannelInfoToJson(this);
}
