// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'requests_cut_request.g.dart';

@JsonSerializable()
class RequestsCutRequest {
  const RequestsCutRequest({
    this.deleteAfterCut,
    this.ends,
    this.starts,
  });
  
  factory RequestsCutRequest.fromJson(Map<String, Object?> json) => _$RequestsCutRequestFromJson(json);
  
  final bool? deleteAfterCut;
  final List<String>? ends;
  final List<String>? starts;

  Map<String, Object?> toJson() => _$RequestsCutRequestToJson(this);
}
