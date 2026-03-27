// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'db_video_preview.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DbVideoPreview _$DbVideoPreviewFromJson(Map<String, dynamic> json) =>
    DbVideoPreview(
      previewPath: json['previewPath'] as String,
      createdAt: json['createdAt'] as String?,
      frameCount: (json['frameCount'] as num?)?.toInt(),
      frameInterval: (json['frameInterval'] as num?)?.toInt(),
      recordingId: (json['recordingId'] as num?)?.toInt(),
      updatedAt: json['updatedAt'] as String?,
      videoPreviewId: (json['videoPreviewId'] as num?)?.toInt(),
    );

Map<String, dynamic> _$DbVideoPreviewToJson(DbVideoPreview instance) =>
    <String, dynamic>{
      'createdAt': instance.createdAt,
      'frameCount': instance.frameCount,
      'frameInterval': instance.frameInterval,
      'previewPath': instance.previewPath,
      'recordingId': instance.recordingId,
      'updatedAt': instance.updatedAt,
      'videoPreviewId': instance.videoPreviewId,
    };
