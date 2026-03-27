// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_recording.dart';

part 'db_channel.g.dart';

@JsonSerializable()
class DbChannel {
  const DbChannel({
    this.channelId,
    this.channelName,
    this.createdAt,
    this.deleted,
    this.displayName,
    this.fav,
    this.isPaused,
    this.minDuration,
    this.recordings,
    this.recordingsCount,
    this.recordingsSize,
    this.skipStart,
    this.tags,
    this.url,
  });
  
  factory DbChannel.fromJson(Map<String, Object?> json) => _$DbChannelFromJson(json);
  
  final int? channelId;
  final String? channelName;
  final String? createdAt;
  final bool? deleted;
  final String? displayName;
  final bool? fav;
  final bool? isPaused;
  final int? minDuration;

  /// 1:n
  final List<DbRecording>? recordings;

  /// Only for query result.
  final int? recordingsCount;
  final int? recordingsSize;
  final int? skipStart;
  final List<String>? tags;
  final String? url;

  Map<String, Object?> toJson() => _$DbChannelToJson(this);
}
