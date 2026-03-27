// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_job_worker_status.g.dart';

@JsonSerializable()
class ResponsesJobWorkerStatus {
  const ResponsesJobWorkerStatus({
    this.isProcessing,
  });
  
  factory ResponsesJobWorkerStatus.fromJson(Map<String, Object?> json) => _$ResponsesJobWorkerStatusFromJson(json);
  
  final bool? isProcessing;

  Map<String, Object?> toJson() => _$ResponsesJobWorkerStatusToJson(this);
}
