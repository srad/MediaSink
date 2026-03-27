// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_jobs_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsJobsRequest _$RequestsJobsRequestFromJson(Map<String, dynamic> json) =>
    RequestsJobsRequest(
      skip: (json['skip'] as num?)?.toInt(),
      sortOrder:
          json['sortOrder'] == null
              ? null
              : DbJobOrder.fromJson(json['sortOrder'] as String),
      states:
          (json['states'] as List<dynamic>?)
              ?.map((e) => DbJobStatus.fromJson(e as String))
              .toList(),
      take: (json['take'] as num?)?.toInt(),
    );

Map<String, dynamic> _$RequestsJobsRequestToJson(
  RequestsJobsRequest instance,
) => <String, dynamic>{
  'skip': instance.skip,
  'sortOrder': _$DbJobOrderEnumMap[instance.sortOrder],
  'states': instance.states?.map((e) => _$DbJobStatusEnumMap[e]!).toList(),
  'take': instance.take,
};

const _$DbJobOrderEnumMap = {
  DbJobOrder.asc: 'ASC',
  DbJobOrder.desc: 'DESC',
  DbJobOrder.$unknown: r'$unknown',
};

const _$DbJobStatusEnumMap = {
  DbJobStatus.completed: 'completed',
  DbJobStatus.open: 'open',
  DbJobStatus.error: 'error',
  DbJobStatus.canceled: 'canceled',
  DbJobStatus.$unknown: r'$unknown',
};
