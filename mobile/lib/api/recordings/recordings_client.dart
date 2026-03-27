// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/database_job.dart';
import '../models/database_recording.dart';
import '../models/requests_cut_request.dart';

part 'recordings_client.g.dart';

@RestApi()
abstract class RecordingsClient {
  factory RecordingsClient(Dio dio, {String? baseUrl}) = _RecordingsClient;

  /// Return a list of recordings.
  ///
  /// Return a list of recordings.
  @GET('/recordings')
  Future<List<DatabaseRecording>> getRecordings();

  /// Returns all bookmarked recordings.
  ///
  /// Returns all bookmarked recordings.
  @GET('/recordings/bookmarks')
  Future<List<DatabaseRecording>> getRecordingsBookmarks();

  /// Get the top N the latest recordings.
  ///
  /// Get the top N the latest recordings.
  ///
  /// [limit] - How many recordings.
  @GET('/recordings/filter/{column}/{order}/{limit}')
  Future<List<DatabaseRecording>> getRecordingsFilterColumnOrderLimit({
    @Path('limit') String? limit,
  });

  /// Return a list of recordings.
  ///
  /// Return a list of recordings.
  @POST('/recordings/generate/posters')
  Future<void> postRecordingsGeneratePosters();

  /// Returns if current the videos are updated.
  ///
  /// Returns if current the videos are updated.
  @GET('/recordings/isupdating')
  Future<void> getRecordingsIsupdating();

  /// Get random videos.
  ///
  /// [limit] - How many recordings.
  @GET('/recordings/random/{limit}')
  Future<List<DatabaseRecording>> getRecordingsRandomLimit({
    @Path('limit') String? limit,
  });

  /// Return a list of recordings.
  ///
  /// Return a list of recordings.
  @POST('/recordings/updateinfo')
  Future<void> postRecordingsUpdateinfo();

  /// Download a file from a channel.
  ///
  /// Download a file from a channel.
  ///
  /// [channelName] - Channel name.
  ///
  /// [filename] - Filename to generate the preview for.
  @GET('/recordings/{channelName}/{filename}/download')
  Future<void> getRecordingsChannelNameFilenameDownload({
    @Path('channelName') required String channelName,
    @Path('filename') required String filename,
  });

  /// Return a list of recordings for a particular channel.
  ///
  /// Return a list of recordings for a particular channel.
  ///
  /// [id] - Recording item id.
  @GET('/recordings/{id}')
  Future<DatabaseRecording> getRecordingsId({
    @Path('id') required int id,
  });

  /// Delete recording.
  ///
  /// Delete recording.
  ///
  /// [id] - Recording item id.
  @DELETE('/recordings/{id}')
  Future<void> deleteRecordingsId({
    @Path('id') required int id,
  });

  /// Cut a video and merge all defined segments.
  ///
  /// Cut a video and merge all defined segments.
  ///
  /// [id] - Recording item id.
  ///
  /// [cutRequest] - Start and end timestamp of cutting sequences.
  @POST('/recordings/{id}/cut')
  Future<DatabaseJob> postRecordingsIdCut({
    @Path('id') required int id,
    @Body() required RequestsCutRequest cutRequest,
  });

  /// Bookmark a certain video in a channel.
  ///
  /// Bookmark a certain video in a channel.
  ///
  /// [id] - Recording item id.
  @PATCH('/recordings/{id}/fav')
  Future<void> patchRecordingsIdFav({
    @Path('id') required int id,
  });

  /// Generate preview for a certain video in a channel.
  ///
  /// Generate preview for a certain video in a channel.
  ///
  /// [id] - Recording item id.
  @POST('/recordings/{id}/preview')
  Future<List<DatabaseJob>> postRecordingsIdPreview({
    @Path('id') required int id,
  });

  /// Bookmark a certain video in a channel.
  ///
  /// Bookmark a certain video in a channel.
  ///
  /// [id] - Recording item id.
  @PATCH('/recordings/{id}/unfav')
  Future<void> patchRecordingsIdUnfav({
    @Path('id') required int id,
  });

  /// Cut a video and merge all defined segments.
  ///
  /// Cut a video and merge all defined segments.
  ///
  /// [id] - Recording item id.
  ///
  /// [mediaType] - Media type to convert to: 720, 1080, mp3.
  @POST('/recordings/{id}/{mediaType}/convert')
  Future<DatabaseJob> postRecordingsIdMediaTypeConvert({
    @Path('id') required int id,
    @Path('mediaType') required String mediaType,
  });
}
