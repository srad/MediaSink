// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_analysis_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesAnalysisResponse _$ResponsesAnalysisResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesAnalysisResponse(
  analysisId: (json['analysisId'] as num?)?.toInt(),
  highlights:
      (json['highlights'] as List<dynamic>?)
          ?.map((e) => DbHighlightInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
  recordingId: (json['recordingId'] as num?)?.toInt(),
  scenes:
      (json['scenes'] as List<dynamic>?)
          ?.map((e) => DbSceneInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
  segments:
      (json['segments'] as List<dynamic>?)
          ?.map((e) => DbSegmentInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
  status: json['status'] as String?,
);

Map<String, dynamic> _$ResponsesAnalysisResponseToJson(
  ResponsesAnalysisResponse instance,
) => <String, dynamic>{
  'analysisId': instance.analysisId,
  'highlights': instance.highlights,
  'recordingId': instance.recordingId,
  'scenes': instance.scenes,
  'segments': instance.segments,
  'status': instance.status,
};
