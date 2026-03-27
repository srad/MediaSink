// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_regenerate_chapters_response.g.dart';

@JsonSerializable()
class ResponsesRegenerateChaptersResponse {
  const ResponsesRegenerateChaptersResponse({
    this.enqueued,
    this.recordings,
    this.removedJobs,
  });
  
  factory ResponsesRegenerateChaptersResponse.fromJson(Map<String, Object?> json) => _$ResponsesRegenerateChaptersResponseFromJson(json);
  
  final int? enqueued;
  final int? recordings;
  final int? removedJobs;

  Map<String, Object?> toJson() => _$ResponsesRegenerateChaptersResponseToJson(this);
}
