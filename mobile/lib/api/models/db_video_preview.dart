// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'db_video_preview.g.dart';

@JsonSerializable()
class DbVideoPreview {
  const DbVideoPreview({
    required this.previewPath,
    this.createdAt,
    this.frameCount,
    this.frameInterval,
    this.recordingId,
    this.updatedAt,
    this.videoPreviewId,
  });
  
  factory DbVideoPreview.fromJson(Map<String, Object?> json) => _$DbVideoPreviewFromJson(json);
  
  final String? createdAt;
  final int? frameCount;
  final int? frameInterval;
  final String previewPath;
  final int? recordingId;
  final String? updatedAt;
  final int? videoPreviewId;

  Map<String, Object?> toJson() => _$DbVideoPreviewToJson(this);
}
