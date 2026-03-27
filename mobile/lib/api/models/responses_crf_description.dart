// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_crf_description.g.dart';

@JsonSerializable()
class ResponsesCrfDescription {
  const ResponsesCrfDescription({
    this.approxRatio,
    this.description,
    this.label,
    this.quality,
    this.value,
  });
  
  factory ResponsesCrfDescription.fromJson(Map<String, Object?> json) => _$ResponsesCrfDescriptionFromJson(json);
  
  final num? approxRatio;
  final String? description;
  final String? label;
  final String? quality;
  final int? value;

  Map<String, Object?> toJson() => _$ResponsesCrfDescriptionToJson(this);
}
