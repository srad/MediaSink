// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_recording.dart';

part 'responses_similar_video_group.g.dart';

@JsonSerializable()
class ResponsesSimilarVideoGroup {
  const ResponsesSimilarVideoGroup({
    this.groupId,
    this.maxSimilarity,
    this.videos,
  });
  
  factory ResponsesSimilarVideoGroup.fromJson(Map<String, Object?> json) => _$ResponsesSimilarVideoGroupFromJson(json);
  
  final int? groupId;
  final num? maxSimilarity;
  final List<DbRecording>? videos;

  Map<String, Object?> toJson() => _$ResponsesSimilarVideoGroupToJson(this);
}
