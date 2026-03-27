// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'db_job.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DbJob _$DbJobFromJson(Map<String, dynamic> json) => DbJob(
  active: json['active'] as bool?,
  args: json['args'] as String?,
  channelId: (json['channelId'] as num?)?.toInt(),
  channelName: json['channelName'] as String?,
  command: json['command'] as String?,
  completedAt: json['completedAt'] as String?,
  createdAt: json['createdAt'] as String?,
  durationMs: (json['durationMs'] as num?)?.toInt(),
  filename: json['filename'] as String?,
  filepath: json['filepath'] as String?,
  info: json['info'] as String?,
  jobId: (json['jobId'] as num?)?.toInt(),
  pid: (json['pid'] as num?)?.toInt(),
  priority:
      json['priority'] == null
          ? null
          : DbJobPriority.fromJson((json['priority'] as num).toInt()),
  progress: json['progress'] as String?,
  recordingId: (json['recordingId'] as num?)?.toInt(),
  startedAt: json['startedAt'] as String?,
  status:
      json['status'] == null
          ? null
          : DbJobStatus.fromJson(json['status'] as String),
  task:
      json['task'] == null ? null : DbJobTask.fromJson(json['task'] as String),
);

Map<String, dynamic> _$DbJobToJson(DbJob instance) => <String, dynamic>{
  'active': instance.active,
  'args': instance.args,
  'channelId': instance.channelId,
  'channelName': instance.channelName,
  'command': instance.command,
  'completedAt': instance.completedAt,
  'createdAt': instance.createdAt,
  'durationMs': instance.durationMs,
  'filename': instance.filename,
  'filepath': instance.filepath,
  'info': instance.info,
  'jobId': instance.jobId,
  'pid': instance.pid,
  'priority': _$DbJobPriorityEnumMap[instance.priority],
  'progress': instance.progress,
  'recordingId': instance.recordingId,
  'startedAt': instance.startedAt,
  'status': _$DbJobStatusEnumMap[instance.status],
  'task': _$DbJobTaskEnumMap[instance.task],
};

const _$DbJobPriorityEnumMap = {
  DbJobPriority.value1: 1,
  DbJobPriority.value3: 3,
  DbJobPriority.value5: 5,
  DbJobPriority.$unknown: r'$unknown',
};

const _$DbJobStatusEnumMap = {
  DbJobStatus.completed: 'completed',
  DbJobStatus.open: 'open',
  DbJobStatus.error: 'error',
  DbJobStatus.canceled: 'canceled',
  DbJobStatus.$unknown: r'$unknown',
};

const _$DbJobTaskEnumMap = {
  DbJobTask.convert: 'convert',
  DbJobTask.previewFrames: 'preview-frames',
  DbJobTask.analyzeFrames: 'analyze-frames',
  DbJobTask.cut: 'cut',
  DbJobTask.merge: 'merge',
  DbJobTask.enhanceVideo: 'enhance-video',
  DbJobTask.$unknown: r'$unknown',
};
