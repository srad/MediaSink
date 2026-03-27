// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_cut_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsCutRequest _$RequestsCutRequestFromJson(Map<String, dynamic> json) =>
    RequestsCutRequest(
      deleteAfterCut: json['deleteAfterCut'] as bool?,
      ends: (json['ends'] as List<dynamic>?)?.map((e) => e as String).toList(),
      starts:
          (json['starts'] as List<dynamic>?)?.map((e) => e as String).toList(),
    );

Map<String, dynamic> _$RequestsCutRequestToJson(RequestsCutRequest instance) =>
    <String, dynamic>{
      'deleteAfterCut': instance.deleteAfterCut,
      'ends': instance.ends,
      'starts': instance.starts,
    };
