// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'services_regeneration_progress.g.dart';

@JsonSerializable()
class ServicesRegenerationProgress {
  const ServicesRegenerationProgress({
    this.current,
    this.currentVideo,
    this.isRunning,
    this.total,
  });
  
  factory ServicesRegenerationProgress.fromJson(Map<String, Object?> json) => _$ServicesRegenerationProgressFromJson(json);
  
  final int? current;
  final String? currentVideo;
  final bool? isRunning;
  final int? total;

  Map<String, Object?> toJson() => _$ServicesRegenerationProgressToJson(this);
}
