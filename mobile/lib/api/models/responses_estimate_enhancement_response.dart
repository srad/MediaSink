// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_estimate_enhancement_response.g.dart';

@JsonSerializable()
class ResponsesEstimateEnhancementResponse {
  const ResponsesEstimateEnhancementResponse({
    this.compressionRatio,
    this.estimatedFileSize,
    this.estimatedFileSizeMb,
    this.inputFileSize,
  });
  
  factory ResponsesEstimateEnhancementResponse.fromJson(Map<String, Object?> json) => _$ResponsesEstimateEnhancementResponseFromJson(json);
  
  final num? compressionRatio;
  final int? estimatedFileSize;
  @JsonKey(name: 'estimatedFileSizeMB')
  final num? estimatedFileSizeMb;
  final int? inputFileSize;

  Map<String, Object?> toJson() => _$ResponsesEstimateEnhancementResponseToJson(this);
}
