// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/db_job.dart';
import '../models/requests_jobs_request.dart';
import '../models/responses_job_worker_status.dart';
import '../models/responses_jobs_response.dart';

part 'jobs_client.g.dart';

@RestApi()
abstract class JobsClient {
  factory JobsClient(Dio dio, {String? baseUrl}) = _JobsClient;

  /// Jobs pagination.
  ///
  /// Allow paging through jobs by providing skip, take, statuses, and sort order.
  ///
  /// [jobsRequest] - Job pagination properties.
  @POST('/jobs/list')
  Future<ResponsesJobsResponse> postJobsList({
    @Body() required RequestsJobsRequest jobsRequest,
  });

  /// Stop job processing worker.
  ///
  /// Pause the background job processing worker.
  @POST('/jobs/pause')
  Future<void> postJobsPause();

  /// Start job processing worker.
  ///
  /// Resume the background job processing worker.
  @POST('/jobs/resume')
  Future<void> postJobsResume();

  /// Interrupt job gracefully.
  ///
  /// Interrupt a running job by process ID.
  ///
  /// [pid] - Process ID.
  @POST('/jobs/stop/{pid}')
  Future<void> postJobsStopPid({
    @Path('pid') required int pid,
  });

  /// Get job worker status.
  ///
  /// Get the current job processing worker status.
  @GET('/jobs/worker')
  Future<ResponsesJobWorkerStatus> getJobsWorker();

  /// Enqueue a preview job.
  ///
  /// Enqueue a preview job for a video in a channel. For now only preview jobs allowed via REST.
  ///
  /// [id] - Recording item id.
  @POST('/jobs/{id}')
  Future<List<DbJob>> postJobsId({
    @Path('id') required String id,
  });

  /// Interrupt and delete job gracefully.
  ///
  /// Interrupt a running job and remove it from the queue.
  ///
  /// [id] - Job id.
  @DELETE('/jobs/{id}')
  Future<void> deleteJobsId({
    @Path('id') required int id,
  });
}

