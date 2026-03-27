// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:json_annotation/json_annotation.dart';

import 'database_job_status.dart';
import 'database_job_task.dart';

part 'database_job.g.dart';

@JsonSerializable()
class DatabaseJob {
  const DatabaseJob({
    this.active,
    this.args,
    this.channelId,
    this.channelName,
    this.command,
    this.completedAt,
    this.createdAt,
    this.filename,
    this.filepath,
    this.info,
    this.jobId,
    this.pid,
    this.progress,
    this.recordingId,
    this.startedAt,
    this.status,
    this.task,
  });
  
  factory DatabaseJob.fromJson(Map<String, Object?> json) => _$DatabaseJobFromJson(json);
  
  final bool? active;
  final String? args;
  final int? channelId;

  /// Unique entry, this is the actual primary key
  final String? channelName;
  final String? command;
  final String? completedAt;
  final String? createdAt;
  final String? filename;
  final String? filepath;
  final String? info;
  final int? jobId;

  /// Additional information
  final int? pid;
  final String? progress;
  final int? recordingId;
  final String? startedAt;
  final DatabaseJobStatus? status;

  /// Default values only not to break migrations.
  final DatabaseJobTask? task;

  Map<String, Object?> toJson() => _$DatabaseJobToJson(this);
}
