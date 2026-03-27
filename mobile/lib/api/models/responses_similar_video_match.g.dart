// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_similar_video_match.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesSimilarVideoMatch _$ResponsesSimilarVideoMatchFromJson(
  Map<String, dynamic> json,
) => ResponsesSimilarVideoMatch(
  bestTimestamp: json['bestTimestamp'] as num?,
  recording:
      json['recording'] == null
          ? null
          : DbRecording.fromJson(json['recording'] as Map<String, dynamic>),
  similarity: json['similarity'] as num?,
);

Map<String, dynamic> _$ResponsesSimilarVideoMatchToJson(
  ResponsesSimilarVideoMatch instance,
) => <String, dynamic>{
  'bestTimestamp': instance.bestTimestamp,
  'recording': instance.recording,
  'similarity': instance.similarity,
};
