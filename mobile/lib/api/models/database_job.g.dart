// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'database_job.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DatabaseJob _$DatabaseJobFromJson(Map<String, dynamic> json) => DatabaseJob(
  active: json['active'] as bool?,
  args: json['args'] as String?,
  channelId: (json['channelId'] as num?)?.toInt(),
  channelName: json['channelName'] as String?,
  command: json['command'] as String?,
  completedAt: json['completedAt'] as String?,
  createdAt: json['createdAt'] as String?,
  filename: json['filename'] as String?,
  filepath: json['filepath'] as String?,
  info: json['info'] as String?,
  jobId: (json['jobId'] as num?)?.toInt(),
  pid: (json['pid'] as num?)?.toInt(),
  progress: json['progress'] as String?,
  recordingId: (json['recordingId'] as num?)?.toInt(),
  startedAt: json['startedAt'] as String?,
  status:
      json['status'] == null
          ? null
          : DatabaseJobStatus.fromJson(json['status'] as String),
  task:
      json['task'] == null
          ? null
          : DatabaseJobTask.fromJson(json['task'] as String),
);

Map<String, dynamic> _$DatabaseJobToJson(DatabaseJob instance) =>
    <String, dynamic>{
      'active': instance.active,
      'args': instance.args,
      'channelId': instance.channelId,
      'channelName': instance.channelName,
      'command': instance.command,
      'completedAt': instance.completedAt,
      'createdAt': instance.createdAt,
      'filename': instance.filename,
      'filepath': instance.filepath,
      'info': instance.info,
      'jobId': instance.jobId,
      'pid': instance.pid,
      'progress': instance.progress,
      'recordingId': instance.recordingId,
      'startedAt': instance.startedAt,
      'status': _$DatabaseJobStatusEnumMap[instance.status],
      'task': _$DatabaseJobTaskEnumMap[instance.task],
    };

const _$DatabaseJobStatusEnumMap = {
  DatabaseJobStatus.completed: 'completed',
  DatabaseJobStatus.open: 'open',
  DatabaseJobStatus.error: 'error',
  DatabaseJobStatus.canceled: 'canceled',
  DatabaseJobStatus.$unknown: r'$unknown',
};

const _$DatabaseJobTaskEnumMap = {
  DatabaseJobTask.convert: 'convert',
  DatabaseJobTask.previewCover: 'preview-cover',
  DatabaseJobTask.previewStripe: 'preview-stripe',
  DatabaseJobTask.previewVideo: 'preview-video',
  DatabaseJobTask.cut: 'cut',
  DatabaseJobTask.$unknown: r'$unknown',
};
