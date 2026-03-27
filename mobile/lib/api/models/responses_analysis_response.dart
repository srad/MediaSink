// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_highlight_info.dart';
import 'db_scene_info.dart';
import 'db_segment_info.dart';

part 'responses_analysis_response.g.dart';

@JsonSerializable()
class ResponsesAnalysisResponse {
  const ResponsesAnalysisResponse({
    this.analysisId,
    this.highlights,
    this.recordingId,
    this.scenes,
    this.segments,
    this.status,
  });
  
  factory ResponsesAnalysisResponse.fromJson(Map<String, Object?> json) => _$ResponsesAnalysisResponseFromJson(json);
  
  final int? analysisId;
  final List<DbHighlightInfo>? highlights;
  final int? recordingId;
  final List<DbSceneInfo>? scenes;
  final List<DbSegmentInfo>? segments;
  final String? status;

  Map<String, Object?> toJson() => _$ResponsesAnalysisResponseToJson(this);
}
