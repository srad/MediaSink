// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'requests_authentication_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RequestsAuthenticationRequest _$RequestsAuthenticationRequestFromJson(
  Map<String, dynamic> json,
) => RequestsAuthenticationRequest(
  password: json['password'] as String?,
  username: json['username'] as String?,
);

Map<String, dynamic> _$RequestsAuthenticationRequestToJson(
  RequestsAuthenticationRequest instance,
) => <String, dynamic>{
  'password': instance.password,
  'username': instance.username,
};
