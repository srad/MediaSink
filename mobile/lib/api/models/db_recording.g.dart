// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'db_recording.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DbRecording _$DbRecordingFromJson(Map<String, dynamic> json) => DbRecording(
  channelName: json['channelName'] as String,
  filename: json['filename'] as String,
  pathRelative: json['pathRelative'] as String,
  videoType: json['videoType'] as String,
  bitRate: (json['bitRate'] as num?)?.toInt(),
  bookmark: json['bookmark'] as bool?,
  channelId: (json['channelId'] as num?)?.toInt(),
  createdAt: json['createdAt'] as String?,
  duration: json['duration'] as num?,
  height: (json['height'] as num?)?.toInt(),
  packets: (json['packets'] as num?)?.toInt(),
  recordingId: (json['recordingId'] as num?)?.toInt(),
  size: (json['size'] as num?)?.toInt(),
  videoPreview:
      json['videoPreview'] == null
          ? null
          : DbVideoPreview.fromJson(
            json['videoPreview'] as Map<String, dynamic>,
          ),
  width: (json['width'] as num?)?.toInt(),
);

Map<String, dynamic> _$DbRecordingToJson(DbRecording instance) =>
    <String, dynamic>{
      'bitRate': instance.bitRate,
      'bookmark': instance.bookmark,
      'channelId': instance.channelId,
      'channelName': instance.channelName,
      'createdAt': instance.createdAt,
      'duration': instance.duration,
      'filename': instance.filename,
      'height': instance.height,
      'packets': instance.packets,
      'pathRelative': instance.pathRelative,
      'recordingId': instance.recordingId,
      'size': instance.size,
      'videoPreview': instance.videoPreview,
      'videoType': instance.videoType,
      'width': instance.width,
    };
