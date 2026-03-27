// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_regenerate_chapters_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesRegenerateChaptersResponse
_$ResponsesRegenerateChaptersResponseFromJson(Map<String, dynamic> json) =>
    ResponsesRegenerateChaptersResponse(
      enqueued: (json['enqueued'] as num?)?.toInt(),
      recordings: (json['recordings'] as num?)?.toInt(),
      removedJobs: (json['removedJobs'] as num?)?.toInt(),
    );

Map<String, dynamic> _$ResponsesRegenerateChaptersResponseToJson(
  ResponsesRegenerateChaptersResponse instance,
) => <String, dynamic>{
  'enqueued': instance.enqueued,
  'recordings': instance.recordings,
  'removedJobs': instance.removedJobs,
};
