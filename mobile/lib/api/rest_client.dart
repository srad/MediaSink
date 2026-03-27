// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';

import 'admin/admin_client.dart';
import 'analysis/analysis_client.dart';
import 'auth/auth_client.dart';
import 'channels/channels_client.dart';
import 'info/info_client.dart';
import 'jobs/jobs_client.dart';
import 'previews/previews_client.dart';
import 'processes/processes_client.dart';
import 'recorder/recorder_client.dart';
import 'user/user_client.dart';
import 'videos/videos_client.dart';

class RestClient {
  RestClient(
    Dio dio, {
    String? baseUrl,
  })  : _dio = dio,
        _baseUrl = baseUrl;

  final Dio _dio;
  final String? _baseUrl;

  static String get version => '';

  AdminClient? _admin;
  AnalysisClient? _analysis;
  AuthClient? _auth;
  ChannelsClient? _channels;
  InfoClient? _info;
  JobsClient? _jobs;
  PreviewsClient? _previews;
  ProcessesClient? _processes;
  RecorderClient? _recorder;
  UserClient? _user;
  VideosClient? _videos;

  AdminClient get admin => _admin ??= AdminClient(_dio, baseUrl: _baseUrl);

  AnalysisClient get analysis => _analysis ??= AnalysisClient(_dio, baseUrl: _baseUrl);

  AuthClient get auth => _auth ??= AuthClient(_dio, baseUrl: _baseUrl);

  ChannelsClient get channels => _channels ??= ChannelsClient(_dio, baseUrl: _baseUrl);

  InfoClient get info => _info ??= InfoClient(_dio, baseUrl: _baseUrl);

  JobsClient get jobs => _jobs ??= JobsClient(_dio, baseUrl: _baseUrl);

  PreviewsClient get previews => _previews ??= PreviewsClient(_dio, baseUrl: _baseUrl);

  ProcessesClient get processes => _processes ??= ProcessesClient(_dio, baseUrl: _baseUrl);

  RecorderClient get recorder => _recorder ??= RecorderClient(_dio, baseUrl: _baseUrl);

  UserClient get user => _user ??= UserClient(_dio, baseUrl: _baseUrl);

  VideosClient get videos => _videos ??= VideosClient(_dio, baseUrl: _baseUrl);
}
