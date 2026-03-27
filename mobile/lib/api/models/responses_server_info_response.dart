// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_server_info_response.g.dart';

@JsonSerializable()
class ResponsesServerInfoResponse {
  const ResponsesServerInfoResponse({
    this.commit,
    this.version,
  });
  
  factory ResponsesServerInfoResponse.fromJson(Map<String, Object?> json) => _$ResponsesServerInfoResponseFromJson(json);
  
  final String? commit;
  final String? version;

  Map<String, Object?> toJson() => _$ResponsesServerInfoResponseToJson(this);
}
