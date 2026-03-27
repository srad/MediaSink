// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'database_recording.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DatabaseRecording _$DatabaseRecordingFromJson(Map<String, dynamic> json) =>
    DatabaseRecording(
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
      previewCover: json['previewCover'] as String?,
      previewStripe: json['previewStripe'] as String?,
      previewVideo: json['previewVideo'] as String?,
      recordingId: (json['recordingId'] as num?)?.toInt(),
      size: (json['size'] as num?)?.toInt(),
      width: (json['width'] as num?)?.toInt(),
    );

Map<String, dynamic> _$DatabaseRecordingToJson(DatabaseRecording instance) =>
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
      'previewCover': instance.previewCover,
      'previewStripe': instance.previewStripe,
      'previewVideo': instance.previewVideo,
      'recordingId': instance.recordingId,
      'size': instance.size,
      'videoType': instance.videoType,
      'width': instance.width,
    };
