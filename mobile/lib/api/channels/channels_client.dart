// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/db_channel.dart';
import '../models/db_job.dart';
import '../models/db_recording.dart';
import '../models/requests_channel_request.dart';
import '../models/requests_channel_tags_update_request.dart';
import '../models/requests_merge_request.dart';
import '../models/services_channel_info.dart';

part 'channels_client.g.dart';

@RestApi()
abstract class ChannelsClient {
  factory ChannelsClient(Dio dio, {String? baseUrl}) = _ChannelsClient;

  /// Return a list of channels.
  ///
  /// Return a list of channels.
  @GET('/channels')
  Future<List<ServicesChannelInfo>> getChannels();

  /// Add a new channel.
  ///
  /// Add a new channel.
  ///
  /// [channelRequest] - Channel data.
  @POST('/channels')
  Future<ServicesChannelInfo> postChannels({
    @Body() required RequestsChannelRequest channelRequest,
  });

  /// Return the data of one channel.
  ///
  /// Return the data of one channel.
  ///
  /// [id] - Channel id.
  @GET('/channels/{id}')
  Future<ServicesChannelInfo> getChannelsId({
    @Path('id') required int id,
  });

  /// Delete channel.
  ///
  /// Delete a channel and all its associated recordings.
  ///
  /// [id] - Channel id.
  @DELETE('/channels/{id}')
  Future<void> deleteChannelsId({
    @Path('id') required int id,
  });

  /// Update channel data.
  ///
  /// Update channel data.
  ///
  /// [id] - Channel id.
  ///
  /// [channelRequest] - Channel data.
  @PATCH('/channels/{id}')
  Future<DbChannel> patchChannelsId({
    @Path('id') required int id,
    @Body() required RequestsChannelRequest channelRequest,
  });

  /// Bookmark a channel.
  ///
  /// Mark a channel as favorite/bookmarked.
  ///
  /// [id] - Channel id.
  @PATCH('/channels/{id}/fav')
  Future<void> patchChannelsIdFav({
    @Path('id') required int id,
  });

  /// Merge multiple videos.
  ///
  /// Merge multiple videos with optional re-encoding to highest quality spec.
  ///
  /// [id] - Channel id.
  ///
  /// [mergeRequest] - Recording IDs and merge options.
  @POST('/channels/{id}/merge')
  Future<DbJob> postChannelsIdMerge({
    @Path('id') required int id,
    @Body() required RequestsMergeRequest mergeRequest,
  });

  /// Pause channel recording.
  ///
  /// Pause/stop recording for a channel.
  ///
  /// [id] - Channel id.
  @POST('/channels/{id}/pause')
  Future<void> postChannelsIdPause({
    @Path('id') required int id,
  });

  /// Resume channel recording.
  ///
  /// Resume/restart recording for a channel that was paused.
  ///
  /// [id] - Channel id.
  @POST('/channels/{id}/resume')
  Future<void> postChannelsIdResume({
    @Path('id') required int id,
  });

  /// Tag a channel.
  ///
  /// Tag a channel.
  ///
  /// [channelTagsUpdateRequest] - Channel data.
  ///
  /// [id] - Channel id.
  @PATCH('/channels/{id}/tags')
  Future<void> patchChannelsIdTags({
    @Body() required RequestsChannelTagsUpdateRequest channelTagsUpdateRequest,
    @Path('id') required int id,
  });

  /// Remove channel from bookmarks.
  ///
  /// Remove a channel from favorites/bookmarks.
  ///
  /// [id] - Channel id.
  @PATCH('/channels/{id}/unfav')
  Future<void> patchChannelsIdUnfav({
    @Path('id') required int id,
  });

  /// Upload video file to channel.
  ///
  /// Upload a video file to a channel's recordings.
  ///
  /// [id] - Channel id.
  ///
  /// [file] - Video file to upload.
  @MultiPart()
  @POST('/channels/{id}/upload')
  Future<DbRecording> postChannelsIdUpload({
    @Path('id') required int id,
    @Part(name: 'file') required File file,
  });
}

