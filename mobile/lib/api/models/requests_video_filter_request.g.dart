// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_video_filter_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsVideoFilterRequest _$RequestsVideoFilterRequestFromJson(
  Map<String, dynamic> json,
) => RequestsVideoFilterRequest(
  skip: (json['skip'] as num?)?.toInt(),
  sortColumn:
      json['sortColumn'] == null
          ? null
          : RequestsVideoSortColumn.fromJson(json['sortColumn'] as String),
  sortOrder:
      json['sortOrder'] == null
          ? null
          : ModelsSortOrder.fromJson(json['sortOrder'] as String),
  take: (json['take'] as num?)?.toInt(),
);

Map<String, dynamic> _$RequestsVideoFilterRequestToJson(
  RequestsVideoFilterRequest instance,
) => <String, dynamic>{
  'skip': instance.skip,
  'sortColumn': _$RequestsVideoSortColumnEnumMap[instance.sortColumn],
  'sortOrder': _$ModelsSortOrderEnumMap[instance.sortOrder],
  'take': instance.take,
};

const _$RequestsVideoSortColumnEnumMap = {
  RequestsVideoSortColumn.createdAt: 'created_at',
  RequestsVideoSortColumn.size: 'size',
  RequestsVideoSortColumn.duration: 'duration',
  RequestsVideoSortColumn.$unknown: r'$unknown',
};

const _$ModelsSortOrderEnumMap = {
  ModelsSortOrder.asc: 'asc',
  ModelsSortOrder.desc: 'desc',
  ModelsSortOrder.$unknown: r'$unknown',
};
