// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/services_regeneration_progress.dart';

part 'previews_client.g.dart';

@RestApi()
abstract class PreviewsClient {
  factory PreviewsClient(Dio dio, {String? baseUrl}) = _PreviewsClient;

  /// Get preview regeneration progress.
  ///
  /// Get the current progress of preview frame regeneration.
  @GET('/previews/regenerate')
  Future<ServicesRegenerationProgress> getPreviewsRegenerate();

  /// Regenerate all preview frames.
  ///
  /// Delete and regenerate preview frames for all recordings. Runs in background and provides progress updates via WebSocket.
  @POST('/previews/regenerate')
  Future<void> postPreviewsRegenerate();
}

