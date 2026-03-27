// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'responses_resolution_description.g.dart';

@JsonSerializable()
class ResponsesResolutionDescription {
  const ResponsesResolutionDescription({
    this.description,
    this.dimensions,
    this.resolution,
    this.useCase,
  });
  
  factory ResponsesResolutionDescription.fromJson(Map<String, Object?> json) => _$ResponsesResolutionDescriptionFromJson(json);
  
  final String? description;
  final String? dimensions;
  final String? resolution;
  final String? useCase;

  Map<String, Object?> toJson() => _$ResponsesResolutionDescriptionToJson(this);
}
