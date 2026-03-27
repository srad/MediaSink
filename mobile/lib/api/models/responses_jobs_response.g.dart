// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_jobs_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesJobsResponse _$ResponsesJobsResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesJobsResponse(
  jobs:
      (json['jobs'] as List<dynamic>?)
          ?.map((e) => DbJob.fromJson(e as Map<String, dynamic>))
          .toList(),
  skip: (json['skip'] as num?)?.toInt(),
  take: (json['take'] as num?)?.toInt(),
  totalCount: (json['totalCount'] as num?)?.toInt(),
);

Map<String, dynamic> _$ResponsesJobsResponseToJson(
  ResponsesJobsResponse instance,
) => <String, dynamic>{
  'jobs': instance.jobs,
  'skip': instance.skip,
  'take': instance.take,
  'totalCount': instance.totalCount,
};
