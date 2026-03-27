// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/responses_recording_status_response.dart';

part 'recorder_client.g.dart';

@RestApi()
abstract class RecorderClient {
  factory RecorderClient(Dio dio, {String? baseUrl}) = _RecorderClient;

  /// Get recorder status.
  ///
  /// Get the current recording/streaming recorder status.
  @GET('/recorder')
  Future<ResponsesRecordingStatusResponse> getRecorder();

  /// Pause the recorder.
  ///
  /// Stop/pause the recording and streaming recorder.
  @POST('/recorder/pause')
  Future<void> postRecorderPause();

  /// Resume the recorder.
  ///
  /// Resume/restart the recording and streaming recorder.
  @POST('/recorder/resume')
  Future<void> postRecorderResume();
}

