// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_video_preview.dart';

part 'db_recording.g.dart';

@JsonSerializable()
class DbRecording {
  const DbRecording({
    required this.channelName,
    required this.filename,
    required this.pathRelative,
    required this.videoType,
    this.bitRate,
    this.bookmark,
    this.channelId,
    this.createdAt,
    this.duration,
    this.height,
    this.packets,
    this.recordingId,
    this.size,
    this.videoPreview,
    this.width,
  });
  
  factory DbRecording.fromJson(Map<String, Object?> json) => _$DbRecordingFromJson(json);
  
  final int? bitRate;
  final bool? bookmark;
  final int? channelId;
  final String channelName;
  final String? createdAt;
  final num? duration;
  final String filename;
  final int? height;

  /// Total number of video packets/frames.
  final int? packets;
  final String pathRelative;
  final int? recordingId;
  final int? size;
  final DbVideoPreview? videoPreview;
  final String videoType;
  final int? width;

  Map<String, Object?> toJson() => _$DbRecordingToJson(this);
}
