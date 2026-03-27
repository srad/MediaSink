// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_preview_manifest_response.g.dart';

@JsonSerializable()
class ResponsesPreviewManifestResponse {
  const ResponsesPreviewManifestResponse({
    this.frameCount,
    this.previewPath,
    this.recordingId,
    this.timestamps,
  });
  
  factory ResponsesPreviewManifestResponse.fromJson(Map<String, Object?> json) => _$ResponsesPreviewManifestResponseFromJson(json);
  
  final int? frameCount;
  final String? previewPath;
  final int? recordingId;
  final List<int>? timestamps;

  Map<String, Object?> toJson() => _$ResponsesPreviewManifestResponseToJson(this);
}
