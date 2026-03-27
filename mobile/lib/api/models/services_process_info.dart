// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'services_process_info.g.dart';

@JsonSerializable()
class ServicesProcessInfo {
  const ServicesProcessInfo({
    this.args,
    this.id,
    this.output,
    this.path,
    this.pid,
  });
  
  factory ServicesProcessInfo.fromJson(Map<String, Object?> json) => _$ServicesProcessInfoFromJson(json);
  
  final String? args;
  final int? id;
  final String? output;
  final String? path;
  final int? pid;

  Map<String, Object?> toJson() => _$ServicesProcessInfoToJson(this);
}
