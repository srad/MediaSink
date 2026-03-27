// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_import_info_response.g.dart';

@JsonSerializable()
class ResponsesImportInfoResponse {
  const ResponsesImportInfoResponse({
    this.isImporting,
    this.progress,
    this.size,
  });
  
  factory ResponsesImportInfoResponse.fromJson(Map<String, Object?> json) => _$ResponsesImportInfoResponseFromJson(json);
  
  final bool? isImporting;
  final int? progress;
  final int? size;

  Map<String, Object?> toJson() => _$ResponsesImportInfoResponseToJson(this);
}
