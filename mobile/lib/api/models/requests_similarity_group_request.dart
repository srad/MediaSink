// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'requests_similarity_group_request.g.dart';

@JsonSerializable()
class RequestsSimilarityGroupRequest {
  const RequestsSimilarityGroupRequest({
    this.includeSingletons,
    this.pairLimit,
    this.recordingIds,
    this.similarity,
  });
  
  factory RequestsSimilarityGroupRequest.fromJson(Map<String, Object?> json) => _$RequestsSimilarityGroupRequestFromJson(json);
  
  /// Include singleton groups (recordings without neighbors above threshold).
  final bool? includeSingletons;

  /// Hard cap on pairwise comparisons/edges considered.
  final int? pairLimit;

  /// Optional subset. When empty, all recordings with frame vectors are considered.
  final List<int>? recordingIds;

  /// Similarity threshold. Supports 0..1 and 0..100 (percent) formats.
  final num? similarity;

  Map<String, Object?> toJson() => _$RequestsSimilarityGroupRequestToJson(this);
}
