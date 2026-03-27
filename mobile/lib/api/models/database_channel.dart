// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

import 'database_recording.dart';

part 'database_channel.g.dart';

@JsonSerializable()
class DatabaseChannel {
  const DatabaseChannel({
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
  
  factory DatabaseChannel.fromJson(Map<String, Object?> json) => _$DatabaseChannelFromJson(json);
  
  final int? channelId;
  final String? channelName;
  final String? createdAt;
  final bool? deleted;
  final String? displayName;
  final bool? fav;
  final bool? isPaused;
  final int? minDuration;

  /// 1:n
  final List<DatabaseRecording>? recordings;

  /// Only for query result.
  final int? recordingsCount;
  final int? recordingsSize;
  final int? skipStart;
  final List<String>? tags;
  final String? url;

  Map<String, Object?> toJson() => _$DatabaseChannelToJson(this);
}
