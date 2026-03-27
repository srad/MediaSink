// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/requests_authentication_request.dart';
import '../models/responses_login_response.dart';

part 'auth_client.g.dart';

@RestApi()
abstract class AuthClient {
  factory AuthClient(Dio dio, {String? baseUrl}) = _AuthClient;

  /// User login.
  ///
  /// User login.
  ///
  /// [authenticationRequest] - Username and password.
  @POST('/auth/login')
  Future<ResponsesLoginResponse> postAuthLogin({
    @Body() required RequestsAuthenticationRequest authenticationRequest,
  });

  /// User logout.
  ///
  /// User logout, clears the authentication session.
  @POST('/auth/logout')
  Future<void> postAuthLogout();

  /// Create new user account.
  ///
  /// Create a new user account with username and password.
  ///
  /// [authenticationRequest] - Username and password.
  @POST('/auth/signup')
  Future<void> postAuthSignup({
    @Body() required RequestsAuthenticationRequest authenticationRequest,
  });
}

