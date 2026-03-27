// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:json_annotation/json_annotation.dart';

part 'requests_authentication_request.g.dart';

@JsonSerializable()
class RequestsAuthenticationRequest {
  const RequestsAuthenticationRequest({
    this.password,
    this.username,
  });
  
  factory RequestsAuthenticationRequest.fromJson(Map<String, Object?> json) => _$RequestsAuthenticationRequestFromJson(json);
  
  final String? password;
  final String? username;

  Map<String, Object?> toJson() => _$RequestsAuthenticationRequestToJson(this);
}
