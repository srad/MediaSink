// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

part 'database_recording.g.dart';

@JsonSerializable()
class DatabaseRecording {
  const DatabaseRecording({
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
    this.previewCover,
    this.previewStripe,
    this.previewVideo,
    this.recordingId,
    this.size,
    this.width,
  });
  
  factory DatabaseRecording.fromJson(Map<String, Object?> json) => _$DatabaseRecordingFromJson(json);
  
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
  final String? previewCover;
  final String? previewStripe;
  final String? previewVideo;
  final int? recordingId;
  final int? size;
  final String videoType;
  final int? width;

  Map<String, Object?> toJson() => _$DatabaseRecordingToJson(this);
}
