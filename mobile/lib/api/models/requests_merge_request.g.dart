// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_merge_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsMergeRequest _$RequestsMergeRequestFromJson(
  Map<String, dynamic> json,
) => RequestsMergeRequest(
  recordingIds:
      (json['recordingIds'] as List<dynamic>)
          .map((e) => (e as num).toInt())
          .toList(),
  reEncode: json['reEncode'] as bool?,
);

Map<String, dynamic> _$RequestsMergeRequestToJson(
  RequestsMergeRequest instance,
) => <String, dynamic>{
  'reEncode': instance.reEncode,
  'recordingIds': instance.recordingIds,
};
