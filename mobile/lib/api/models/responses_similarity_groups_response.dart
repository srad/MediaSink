// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'responses_similar_video_group.dart';

part 'responses_similarity_groups_response.g.dart';

@JsonSerializable()
class ResponsesSimilarityGroupsResponse {
  const ResponsesSimilarityGroupsResponse({
    this.analyzedCount,
    this.groupCount,
    this.groups,
    this.similarityThreshold,
  });
  
  factory ResponsesSimilarityGroupsResponse.fromJson(Map<String, Object?> json) => _$ResponsesSimilarityGroupsResponseFromJson(json);
  
  /// recordings with stored frame vectors
  final int? analyzedCount;
  final int? groupCount;
  final List<ResponsesSimilarVideoGroup>? groups;
  final num? similarityThreshold;

  Map<String, Object?> toJson() => _$ResponsesSimilarityGroupsResponseToJson(this);
}
