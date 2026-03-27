// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/responses_import_info_response.dart';
import '../models/responses_regenerate_chapters_response.dart';
import '../models/responses_server_info_response.dart';

part 'admin_client.g.dart';

@RestApi()
abstract class AdminClient {
  factory AdminClient(Dio dio, {String? baseUrl}) = _AdminClient;

  /// Remove stale chapter jobs and enqueue fresh analysis for all recordings.
  ///
  /// Deletes existing analyze-frames jobs and creates a fresh chapter-analysis job for every recording.
  @POST('/admin/chapters/regenerate')
  Future<ResponsesRegenerateChaptersResponse> postAdminChaptersRegenerate();

  /// Returns current import progress information.
  ///
  /// Get the current import progress status and information.
  @GET('/admin/import')
  Future<ResponsesImportInfoResponse> getAdminImport();

  /// Run once the import of mp4 files in the recordings folder.
  ///
  /// Import all mp4 files in the recordings directory that are not yet in the system database.
  @POST('/admin/import')
  Future<void> postAdminImport();

  /// Returns server version information.
  ///
  /// version information.
  @GET('/admin/version')
  Future<ResponsesServerInfoResponse> getAdminVersion();
}

