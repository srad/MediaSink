// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'responses_import_info_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResponsesImportInfoResponse _$ResponsesImportInfoResponseFromJson(
  Map<String, dynamic> json,
) => ResponsesImportInfoResponse(
  isImporting: json['isImporting'] as bool?,
  progress: (json['progress'] as num?)?.toInt(),
  size: (json['size'] as num?)?.toInt(),
);

Map<String, dynamic> _$ResponsesImportInfoResponseToJson(
  ResponsesImportInfoResponse instance,
) => <String, dynamic>{
  'isImporting': instance.isImporting,
  'progress': instance.progress,
  'size': instance.size,
};
