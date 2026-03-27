// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_similarity_groups_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesSimilarityGroupsResponse _$ResponsesSimilarityGroupsResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesSimilarityGroupsResponse(
  analyzedCount: (json['analyzedCount'] as num?)?.toInt(),
  groupCount: (json['groupCount'] as num?)?.toInt(),
  groups:
      (json['groups'] as List<dynamic>?)
          ?.map(
            (e) =>
                ResponsesSimilarVideoGroup.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
  similarityThreshold: json['similarityThreshold'] as num?,
);

Map<String, dynamic> _$ResponsesSimilarityGroupsResponseToJson(
  ResponsesSimilarityGroupsResponse instance,
) => <String, dynamic>{
  'analyzedCount': instance.analyzedCount,
  'groupCount': instance.groupCount,
  'groups': instance.groups,
  'similarityThreshold': instance.similarityThreshold,
};
