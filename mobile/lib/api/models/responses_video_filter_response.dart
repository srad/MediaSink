// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_recording.dart';

part 'responses_video_filter_response.g.dart';

@JsonSerializable()
class ResponsesVideoFilterResponse {
  const ResponsesVideoFilterResponse({
    this.skip,
    this.take,
    this.totalCount,
    this.videos,
  });
  
  factory ResponsesVideoFilterResponse.fromJson(Map<String, Object?> json) => _$ResponsesVideoFilterResponseFromJson(json);
  
  final int? skip;
  final int? take;
  final int? totalCount;
  final List<DbRecording>? videos;

  Map<String, Object?> toJson() => _$ResponsesVideoFilterResponseToJson(this);
}
