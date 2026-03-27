// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

import 'db_job_priority.dart';
import 'db_job_status.dart';
import 'db_job_task.dart';

part 'db_job.g.dart';

@JsonSerializable()
class DbJob {
  const DbJob({
    this.active,
    this.args,
    this.channelId,
    this.channelName,
    this.command,
    this.completedAt,
    this.createdAt,
    this.durationMs,
    this.filename,
    this.filepath,
    this.info,
    this.jobId,
    this.pid,
    this.priority,
    this.progress,
    this.recordingId,
    this.startedAt,
    this.status,
    this.task,
  });
  
  factory DbJob.fromJson(Map<String, Object?> json) => _$DbJobFromJson(json);
  
  final bool? active;
  final String? args;
  final int? channelId;

  /// Unique entry, this is the actual primary key
  final String? channelName;
  final String? command;
  final String? completedAt;
  final String? createdAt;

  /// Duration in milliseconds
  final int? durationMs;
  final String? filename;
  final String? filepath;
  final String? info;
  final int? jobId;

  /// Additional information
  final int? pid;
  final DbJobPriority? priority;
  final String? progress;
  final int? recordingId;
  final String? startedAt;
  final DbJobStatus? status;

  /// Default values only not to break migrations.
  final DbJobTask? task;

  Map<String, Object?> toJson() => _$DbJobToJson(this);
}
