// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_enqueue_all_response.g.dart';

@JsonSerializable()
class ResponsesEnqueueAllResponse {
  const ResponsesEnqueueAllResponse({
    this.enqueued,
  });
  
  factory ResponsesEnqueueAllResponse.fromJson(Map<String, Object?> json) => _$ResponsesEnqueueAllResponseFromJson(json);
  
  final int? enqueued;

  Map<String, Object?> toJson() => _$ResponsesEnqueueAllResponseToJson(this);
}
