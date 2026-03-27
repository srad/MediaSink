// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_similar_video_group.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesSimilarVideoGroup _$ResponsesSimilarVideoGroupFromJson(
  Map<String, dynamic> json,
) => ResponsesSimilarVideoGroup(
  groupId: (json['groupId'] as num?)?.toInt(),
  maxSimilarity: json['maxSimilarity'] as num?,
  videos:
      (json['videos'] as List<dynamic>?)
          ?.map((e) => DbRecording.fromJson(e as Map<String, dynamic>))
          .toList(),
);

Map<String, dynamic> _$ResponsesSimilarVideoGroupToJson(
  ResponsesSimilarVideoGroup instance,
) => <String, dynamic>{
  'groupId': instance.groupId,
  'maxSimilarity': instance.maxSimilarity,
  'videos': instance.videos,
};
