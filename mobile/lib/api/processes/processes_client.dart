// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, unused_import, invalid_annotation_target, unnecessary_import

import 'package:dio/dio.dart';
import 'package:retrofit/retrofit.dart';

import '../models/services_process_info.dart';

part 'processes_client.g.dart';

@RestApi()
abstract class ProcessesClient {
  factory ProcessesClient(Dio dio, {String? baseUrl}) = _ProcessesClient;

  /// Return a list of streaming processes.
  ///
  /// Return a list of streaming processes.
  @GET('/processes')
  Future<List<ServicesProcessInfo>> getProcesses();
}
