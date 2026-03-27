// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_server_info_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesServerInfoResponse _$ResponsesServerInfoResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesServerInfoResponse(
  commit: json['commit'] as String?,
  version: json['version'] as String?,
);

Map<String, dynamic> _$ResponsesServerInfoResponseToJson(
  ResponsesServerInfoResponse instance,
) => <String, dynamic>{'commit': instance.commit, 'version': instance.version};
