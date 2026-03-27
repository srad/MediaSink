// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'models_sort_order.dart';
import 'requests_video_sort_column.dart';

part 'requests_video_filter_request.g.dart';

@JsonSerializable()
class RequestsVideoFilterRequest {
  const RequestsVideoFilterRequest({
    this.skip,
    this.sortColumn,
    this.sortOrder,
    this.take,
  });
  
  factory RequestsVideoFilterRequest.fromJson(Map<String, Object?> json) => _$RequestsVideoFilterRequestFromJson(json);
  
  final int? skip;
  final RequestsVideoSortColumn? sortColumn;
  final ModelsSortOrder? sortOrder;
  final int? take;

  Map<String, Object?> toJson() => _$RequestsVideoFilterRequestToJson(this);
}
