// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/db_job.dart';
import '../models/db_recording.dart';
import '../models/requests_cut_request.dart';
import '../models/requests_enhance_request.dart';
import '../models/requests_estimate_enhancement_request.dart';
import '../models/requests_video_filter_request.dart';
import '../models/responses_enhancement_descriptions.dart';
import '../models/responses_estimate_enhancement_response.dart';
import '../models/responses_preview_manifest_response.dart';
import '../models/responses_video_filter_response.dart';

part 'videos_client.g.dart';

@RestApi()
abstract class VideosClient {
  factory VideosClient(Dio dio, {String? baseUrl}) = _VideosClient;

  /// Return a list of videos.
  ///
  /// Return a list of videos.
  @GET('/videos')
  Future<List<DbRecording>> getVideos();

  /// Returns all bookmarked videos.
  ///
  /// Returns all bookmarked videos.
  @GET('/videos/bookmarks')
  Future<List<DbRecording>> getVideosBookmarks();

  /// Get enhancement parameter descriptions.
  ///
  /// Return descriptions for all video enhancement parameters (presets, CRF values, resolutions, filters).
  @GET('/videos/enhance/descriptions')
  Future<ResponsesEnhancementDescriptions> getVideosEnhanceDescriptions();

  /// Get the top N the latest videos.
  ///
  /// Get the top N the latest videos.
  ///
  /// [videoFilterRequest] - Video filter containing column name, sort order and skip and limit.
  @POST('/videos/filter')
  Future<ResponsesVideoFilterResponse> postVideosFilter({
    @Body() required RequestsVideoFilterRequest videoFilterRequest,
  });

  /// Check if video metadata update is in progress.
  ///
  /// Get the status of the video metadata update process.
  @POST('/videos/isupdating')
  Future<bool> postVideosIsupdating();

  /// Get random videos.
  ///
  /// Get a random selection of videos from the system.
  ///
  /// [limit] - Number of random videos to return.
  @GET('/videos/random/{limit}')
  Future<List<DbRecording>> getVideosRandomLimit({
    @Path('limit') required int limit,
  });

  /// Update video metadata information.
  ///
  /// Update metadata information for all videos in the system.
  @POST('/videos/updateinfo')
  Future<void> postVideosUpdateinfo();

  /// Return a list of videos for a particular channel.
  ///
  /// Return a list of videos for a particular channel.
  ///
  /// [id] - videos item id.
  @GET('/videos/{id}')
  Future<DbRecording> getVideosId({
    @Path('id') required int id,
  });

  /// Delete video.
  ///
  /// Delete video.
  ///
  /// [id] - video item id.
  @DELETE('/videos/{id}')
  Future<void> deleteVideosId({
    @Path('id') required int id,
  });

  /// Cut a video and merge all defined segments.
  ///
  /// Cut a video and merge all defined segments.
  ///
  /// [id] - video item id.
  ///
  /// [cutRequest] - Start and end timestamp of cutting sequences.
  @POST('/videos/{id}/cut')
  Future<DbJob> postVideosIdCut({
    @Path('id') required int id,
    @Body() required RequestsCutRequest cutRequest,
  });

  /// Download a video file.
  ///
  /// Download a video file as an attachment.
  ///
  /// [id] - Recording item id.
  @GET('/videos/{id}/download')
  Future<void> getVideosIdDownload({
    @Path('id') required int id,
  });

  /// Enhance video quality.
  ///
  /// Enhance a video with denoising, upscaling, and sharpening.
  ///
  /// [id] - Recording id.
  ///
  /// [enhanceRequest] - Enhancement parameters.
  @POST('/videos/{id}/enhance')
  Future<DbJob> postVideosIdEnhance({
    @Path('id') required int id,
    @Body() required RequestsEnhanceRequest enhanceRequest,
  });

  /// Estimate video enhancement file size.
  ///
  /// Estimate the output file size for video enhancement with given parameters.
  ///
  /// [id] - Recording id.
  ///
  /// [estimateEnhancementRequest] - Enhancement parameters.
  @POST('/videos/{id}/estimate-enhancement')
  Future<ResponsesEstimateEnhancementResponse> postVideosIdEstimateEnhancement({
    @Path('id') required int id,
    @Body() required RequestsEstimateEnhancementRequest estimateEnhancementRequest,
  });

  /// Bookmark a video.
  ///
  /// Bookmark/favorite a video for easy access.
  ///
  /// [id] - video item id.
  @PATCH('/videos/{id}/fav')
  Future<void> patchVideosIdFav({
    @Path('id') required int id,
  });

  /// Generate preview for a certain video in a channel.
  ///
  /// Generate preview for a certain video in a channel.
  ///
  /// [id] - videos item id.
  @POST('/videos/{id}/preview')
  Future<List<DbJob>> postVideosIdPreview({
    @Path('id') required int id,
  });

  /// Return actual preview frame timestamps for a video.
  ///
  /// Return the sorted list of actual preview frame timestamps for a video timeline.
  ///
  /// [id] - video item id.
  @GET('/videos/{id}/preview/manifest')
  Future<ResponsesPreviewManifestResponse> getVideosIdPreviewManifest({
    @Path('id') required int id,
  });

  /// Remove video from bookmarks.
  ///
  /// Remove/unbookmark a video from favorites.
  ///
  /// [id] - video item id.
  @PATCH('/videos/{id}/unfav')
  Future<void> patchVideosIdUnfav({
    @Path('id') required int id,
  });

  /// Cut a video and merge all defined segments.
  ///
  /// Cut a video and merge all defined segments.
  ///
  /// [id] - video item id.
  ///
  /// [mediaType] - Media type to convert to: 720, 1080, mp3.
  @POST('/videos/{id}/{mediaType}/convert')
  Future<DbJob> postVideosIdMediaTypeConvert({
    @Path('id') required int id,
    @Path('mediaType') required String mediaType,
  });
}

