// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/requests_similarity_group_request.dart';
import '../models/responses_analysis_response.dart';
import '../models/responses_enqueue_all_response.dart';
import '../models/responses_similarity_groups_response.dart';
import '../models/responses_visual_search_response.dart';

part 'analysis_client.g.dart';

@RestApi()
abstract class AnalysisClient {
  factory AnalysisClient(Dio dio, {String? baseUrl}) = _AnalysisClient;

  /// Enqueue analysis jobs for all recordings.
  ///
  /// Enqueues a video analysis job for every recording in the library.
  @POST('/analysis/all')
  Future<ResponsesEnqueueAllResponse> postAnalysisAll();

  /// Group visually similar videos.
  ///
  /// Build similarity clusters from analyzed frame vectors.
  ///
  /// [request] - Grouping request.
  @POST('/analysis/group')
  Future<ResponsesSimilarityGroupsResponse> postAnalysisGroup({
    @Body() required RequestsSimilarityGroupRequest request,
  });

  /// Search visually similar videos by image.
  ///
  /// Upload a picture and return visually similar videos using frame embeddings.
  ///
  /// [file] - Query image file.
  ///
  /// [similarity] - Similarity threshold (0..1 or 0..100), default 0.8.
  ///
  /// [limit] - Max results (1..200), default 50.
  @MultiPart()
  @POST('/analysis/search/image')
  Future<ResponsesVisualSearchResponse> postAnalysisSearchImage({
    @Part(name: 'file') required File file,
    @Part(name: 'similarity') num? similarity,
    @Part(name: 'limit') int? limit,
  });

  /// Get video analysis result.
  ///
  /// Get the analysis results (scenes and highlights) for a recording.
  ///
  /// [id] - recording id.
  @GET('/analysis/{id}')
  Future<ResponsesAnalysisResponse> getAnalysisId({
    @Path('id') required int id,
  });

  /// Analyze video frames for scenes and highlights.
  ///
  /// Analyze preview frames to detect scenes and highlights. Runs in background as a job.
  ///
  /// [id] - recording id.
  @POST('/analysis/{id}')
  Future<void> postAnalysisId({
    @Path('id') required int id,
  });
}

