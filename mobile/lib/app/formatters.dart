String formatDateTime(String? raw) {
  if (raw == null || raw.isEmpty) {
    return "Unknown";
  }
  final parsed = DateTime.tryParse(raw);
  if (parsed == null) {
    return raw;
  }
  final local = parsed.toLocal();
  final month = local.month.toString().padLeft(2, "0");
  final day = local.day.toString().padLeft(2, "0");
  final hour = local.hour.toString().padLeft(2, "0");
  final minute = local.minute.toString().padLeft(2, "0");
  return "${local.year}-$month-$day $hour:$minute";
}

String formatBytes(int? bytes) {
  final value = bytes ?? 0;
  if (value < 1024) {
    return "$value B";
  }
  if (value < 1024 * 1024) {
    return "${(value / 1024).toStringAsFixed(1)} KB";
  }
  if (value < 1024 * 1024 * 1024) {
    return "${(value / (1024 * 1024)).toStringAsFixed(1)} MB";
  }
  return "${(value / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB";
}

String formatDuration(num? seconds) {
  final totalSeconds = seconds?.round() ?? 0;
  final hours = totalSeconds ~/ 3600;
  final minutes = (totalSeconds % 3600) ~/ 60;
  final remainingSeconds = totalSeconds % 60;
  if (hours > 0) {
    return "${hours}h ${minutes}m";
  }
  if (minutes > 0) {
    return "${minutes}m ${remainingSeconds}s";
  }
  return "${remainingSeconds}s";
}

String formatPercent(String? progress) {
  if (progress == null || progress.isEmpty) {
    return "-";
  }
  final parsed = double.tryParse(progress);
  if (parsed == null) {
    return progress;
  }
  return "${parsed.toStringAsFixed(0)}%";
}
