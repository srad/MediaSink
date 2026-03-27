import "dart:io";

import "package:dio/dio.dart";

import "media_sink_api.dart";

String friendlyErrorMessage(Object error) {
  if (error is MediaSinkApiException) {
    return error.message;
  }
  if (error is ApiVersionMismatchException) {
    return error.message;
  }
  if (error is DioException) {
    return error.message ?? "Request failed.";
  }
  final raw = error.toString();
  if (error is FormatException) {
    return error.message;
  }
  if (raw.startsWith("Exception: ")) {
    return raw.substring("Exception: ".length);
  }
  return raw;
}

bool isConnectivityError(Object error) {
  if (error is MediaSinkApiException) {
    return error.kind == MediaSinkApiErrorKind.serverUnreachable;
  }
  if (error is DioException) {
    return _isConnectivityType(error.type) || error.error is SocketException || error.error is HttpException;
  }
  final message = error.toString().toLowerCase();
  return message.contains("connection reset") ||
      message.contains("socketexception") ||
      message.contains("timed out") ||
      message.contains("connection refused") ||
      message.contains("connection aborted");
}

bool _isConnectivityType(DioExceptionType type) {
  return type == DioExceptionType.connectionError ||
      type == DioExceptionType.connectionTimeout ||
      type == DioExceptionType.receiveTimeout ||
      type == DioExceptionType.sendTimeout;
}
