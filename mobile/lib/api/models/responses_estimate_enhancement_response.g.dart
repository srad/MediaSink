// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_estimate_enhancement_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesEstimateEnhancementResponse
_$ResponsesEstimateEnhancementResponseFromJson(Map<String, dynamic> json) =>
    ResponsesEstimateEnhancementResponse(
      compressionRatio: json['compressionRatio'] as num?,
      estimatedFileSize: (json['estimatedFileSize'] as num?)?.toInt(),
      estimatedFileSizeMb: json['estimatedFileSizeMB'] as num?,
      inputFileSize: (json['inputFileSize'] as num?)?.toInt(),
    );

Map<String, dynamic> _$ResponsesEstimateEnhancementResponseToJson(
  ResponsesEstimateEnhancementResponse instance,
) => <String, dynamic>{
  'compressionRatio': instance.compressionRatio,
  'estimatedFileSize': instance.estimatedFileSize,
  'estimatedFileSizeMB': instance.estimatedFileSizeMb,
  'inputFileSize': instance.inputFileSize,
};
