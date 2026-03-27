// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_preview_manifest_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesPreviewManifestResponse _$ResponsesPreviewManifestResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesPreviewManifestResponse(
  frameCount: (json['frameCount'] as num?)?.toInt(),
  previewPath: json['previewPath'] as String?,
  recordingId: (json['recordingId'] as num?)?.toInt(),
  timestamps:
      (json['timestamps'] as List<dynamic>?)
          ?.map((e) => (e as num).toInt())
          .toList(),
);

Map<String, dynamic> _$ResponsesPreviewManifestResponseToJson(
  ResponsesPreviewManifestResponse instance,
) => <String, dynamic>{
  'frameCount': instance.frameCount,
  'previewPath': instance.previewPath,
  'recordingId': instance.recordingId,
  'timestamps': instance.timestamps,
};
