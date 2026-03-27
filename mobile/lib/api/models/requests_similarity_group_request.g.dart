// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_similarity_group_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsSimilarityGroupRequest _$RequestsSimilarityGroupRequestFromJson(
  Map<String, dynamic> json,
) => RequestsSimilarityGroupRequest(
  includeSingletons: json['includeSingletons'] as bool?,
  pairLimit: (json['pairLimit'] as num?)?.toInt(),
  recordingIds:
      (json['recordingIds'] as List<dynamic>?)
          ?.map((e) => (e as num).toInt())
          .toList(),
  similarity: json['similarity'] as num?,
);

Map<String, dynamic> _$RequestsSimilarityGroupRequestToJson(
  RequestsSimilarityGroupRequest instance,
) => <String, dynamic>{
  'includeSingletons': instance.includeSingletons,
  'pairLimit': instance.pairLimit,
  'recordingIds': instance.recordingIds,
  'similarity': instance.similarity,
};
