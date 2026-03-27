// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_recording_status_response.g.dart';

@JsonSerializable()
class ResponsesRecordingStatusResponse {
  const ResponsesRecordingStatusResponse({
    this.isRecording,
  });
  
  factory ResponsesRecordingStatusResponse.fromJson(Map<String, Object?> json) => _$ResponsesRecordingStatusResponseFromJson(json);
  
  final bool? isRecording;

  Map<String, Object?> toJson() => _$ResponsesRecordingStatusResponseToJson(this);
}
