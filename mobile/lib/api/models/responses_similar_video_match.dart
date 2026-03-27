// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_recording.dart';

part 'responses_similar_video_match.g.dart';

@JsonSerializable()
class ResponsesSimilarVideoMatch {
  const ResponsesSimilarVideoMatch({
    this.bestTimestamp,
    this.recording,
    this.similarity,
  });
  
  factory ResponsesSimilarVideoMatch.fromJson(Map<String, Object?> json) => _$ResponsesSimilarVideoMatchFromJson(json);
  
  final num? bestTimestamp;
  final DbRecording? recording;
  final num? similarity;

  Map<String, Object?> toJson() => _$ResponsesSimilarVideoMatchToJson(this);
}
