// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'requests_merge_request.g.dart';

@JsonSerializable()
class RequestsMergeRequest {
  const RequestsMergeRequest({
    required this.recordingIds,
    this.reEncode,
  });
  
  factory RequestsMergeRequest.fromJson(Map<String, Object?> json) => _$RequestsMergeRequestFromJson(json);
  
  final bool? reEncode;
  final List<int> recordingIds;

  Map<String, Object?> toJson() => _$RequestsMergeRequestToJson(this);
}
