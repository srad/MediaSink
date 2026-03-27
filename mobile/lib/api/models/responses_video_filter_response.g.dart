// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_video_filter_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesVideoFilterResponse _$ResponsesVideoFilterResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesVideoFilterResponse(
  skip: (json['skip'] as num?)?.toInt(),
  take: (json['take'] as num?)?.toInt(),
  totalCount: (json['totalCount'] as num?)?.toInt(),
  videos:
      (json['videos'] as List<dynamic>?)
          ?.map((e) => DbRecording.fromJson(e as Map<String, dynamic>))
          .toList(),
);

Map<String, dynamic> _$ResponsesVideoFilterResponseToJson(
  ResponsesVideoFilterResponse instance,
) => <String, dynamic>{
  'skip': instance.skip,
  'take': instance.take,
  'totalCount': instance.totalCount,
  'videos': instance.videos,
};
