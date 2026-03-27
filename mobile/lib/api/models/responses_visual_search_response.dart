// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'responses_similar_video_match.dart';

part 'responses_visual_search_response.g.dart';

@JsonSerializable()
class ResponsesVisualSearchResponse {
  const ResponsesVisualSearchResponse({
    this.limit,
    this.results,
    this.similarityThreshold,
  });
  
  factory ResponsesVisualSearchResponse.fromJson(Map<String, Object?> json) => _$ResponsesVisualSearchResponseFromJson(json);
  
  final int? limit;
  final List<ResponsesSimilarVideoMatch>? results;
  final num? similarityThreshold;

  Map<String, Object?> toJson() => _$ResponsesVisualSearchResponseToJson(this);
}
