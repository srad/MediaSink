// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_visual_search_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesVisualSearchResponse _$ResponsesVisualSearchResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesVisualSearchResponse(
  limit: (json['limit'] as num?)?.toInt(),
  results:
      (json['results'] as List<dynamic>?)
          ?.map(
            (e) =>
                ResponsesSimilarVideoMatch.fromJson(e as Map<String, dynamic>),
          )
          .toList(),
  similarityThreshold: json['similarityThreshold'] as num?,
);

Map<String, dynamic> _$ResponsesVisualSearchResponseToJson(
  ResponsesVisualSearchResponse instance,
) => <String, dynamic>{
  'limit': instance.limit,
  'results': instance.results,
  'similarityThreshold': instance.similarityThreshold,
};
