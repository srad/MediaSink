import "package:flutter/material.dart";
import "package:provider/provider.dart";
import "package:timeago/timeago.dart" as timeago;

import "../../api/export.dart";
import "../formatters.dart";
import "../jobs_controller.dart";
import "../media_sink_api.dart";
import "../models.dart";
import "../widgets/inline_error_banner.dart";
import "channel_detail_screen.dart";
import "video_player_page.dart";

class JobsScreen extends StatelessWidget {
  const JobsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<JobsController>();
    final api = context.read<MediaSinkApi>();
    final jobs = controller.pagedJobs;

    return RefreshIndicator(
      onRefresh: controller.refresh,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
        children: <Widget>[
          _JobsHeaderCard(controller: controller, onToggleWorker: () => _confirmWorkerToggle(context, controller)),
          if (controller.loading && jobs.isNotEmpty)
            const Padding(
              padding: EdgeInsets.fromLTRB(8, 0, 8, 10),
              child: LinearProgressIndicator(),
            ),
          if (controller.loading && jobs.isEmpty)
            const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: CircularProgressIndicator()),
            ),
          if (controller.error != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 0, 8, 10),
              child: InlineErrorBanner(message: controller.error!, onRetry: controller.refresh),
            ),
          if (!controller.loading && jobs.isEmpty)
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text("No jobs in this view."),
            ),
          for (final job in jobs)
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: _JobCard(
                tab: controller.tab,
                job: job,
                isDeleting: controller.isDeletingJob(job.jobId),
                onOpenChannel: job.channelId == null
                    ? null
                    : () {
                        Navigator.of(context).push(
                          MaterialPageRoute<void>(builder: (_) => ChannelDetailScreen(channelId: job.channelId!)),
                        );
                      },
                onOpenRecording: job.recordingId == null || job.recordingId! <= 0 ? null : () => _openRecording(context, api, job.recordingId!),
                onInspect: () => _showJobDetails(context, api, job),
                onDelete: () => _confirmDelete(context, controller, job),
              ),
            ),
          if (controller.totalPages > 1) _JobsPaginationBar(controller: controller),
        ],
      ),
    );
  }

  Future<void> _confirmDelete(BuildContext context, JobsController controller, DbJob job) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text("Delete job?"),
        content: Text("Job ${job.jobId ?? "-"} will be removed from the queue/history."),
        actions: <Widget>[
          TextButton(onPressed: () => Navigator.of(dialogContext).pop(false), child: const Text("Cancel")),
          FilledButton(onPressed: () => Navigator.of(dialogContext).pop(true), child: const Text("Delete")),
        ],
      ),
    );

    if (confirmed != true || !context.mounted) {
      return;
    }

    try {
      await controller.deleteJob(job);
    } catch (error) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  Future<void> _confirmWorkerToggle(BuildContext context, JobsController controller) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(controller.workerProcessing ? "Pause job worker?" : "Resume job worker?"),
        content: Text(
          controller.workerProcessing
              ? "Queued jobs will stop moving until the worker is resumed."
              : "Queued jobs will continue once the worker is resumed.",
        ),
        actions: <Widget>[
          TextButton(onPressed: () => Navigator.of(dialogContext).pop(false), child: const Text("Cancel")),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(controller.workerProcessing ? "Pause" : "Resume"),
          ),
        ],
      ),
    );

    if (confirmed != true || !context.mounted) {
      return;
    }

    try {
      await controller.toggleWorker();
    } catch (error) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  Future<void> _openRecording(BuildContext context, MediaSinkApi api, int recordingId) async {
    try {
      final video = await api.getVideo(recordingId);
      if (!context.mounted) {
        return;
      }
      await Navigator.of(context).push<VideoPlayerResult>(
        MaterialPageRoute<VideoPlayerResult>(
          builder: (_) => VideoPlayerPage(title: video.filename, url: api.videoFileUrl(video), video: video),
        ),
      );
    } catch (error) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  Future<void> _showJobDetails(BuildContext context, MediaSinkApi api, DbJob job) async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (sheetContext) {
        final mediaQuery = MediaQuery.of(sheetContext);
        final bottomInset = mediaQuery.viewInsets.bottom > 0 ? mediaQuery.viewInsets.bottom : mediaQuery.viewPadding.bottom;

        return SafeArea(
          top: false,
          bottom: false,
          child: SingleChildScrollView(
            padding: EdgeInsets.fromLTRB(16, 4, 16, bottomInset + 16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                Text("Job ${job.jobId ?? "-"}", style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 4),
                Text("${_formatTask(job.task)} in ${job.channelName ?? "Unknown channel"}", style: Theme.of(sheetContext).textTheme.bodySmall),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: <Widget>[
                    _DetailField(label: "Status", value: _statusLabel(JobsTab.completed, job)),
                    _DetailField(label: "File", value: job.filename ?? "-"),
                    _DetailField(label: "Created", value: formatDateTime(job.createdAt)),
                    _DetailField(label: "Started", value: formatDateTime(job.startedAt)),
                    _DetailField(label: "Completed", value: formatDateTime(job.completedAt)),
                    _DetailField(label: "Duration", value: _formatJobDuration(job)),
                  ],
                ),
                const SizedBox(height: 14),
                _DetailBlock(label: "Error / Info", value: job.info ?? "No error text recorded."),
                const SizedBox(height: 10),
                _DetailBlock(label: "Command", value: job.command ?? "-"),
                const SizedBox(height: 10),
                _DetailBlock(label: "Path", value: job.filepath ?? "-"),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _JobsHeaderCard extends StatelessWidget {
  const _JobsHeaderCard({required this.controller, required this.onToggleWorker});

  final JobsController controller;
  final VoidCallback onToggleWorker;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              Row(
                children: <Widget>[
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text("Jobs", style: Theme.of(context).textTheme.titleLarge),
                        const SizedBox(height: 2),
                        Text(controller.summary, style: Theme.of(context).textTheme.bodySmall),
                      ],
                    ),
                  ),
                  PopupMenuButton<int>(
                    enabled: !controller.loading,
                    onSelected: controller.setRowLimit,
                    itemBuilder: (context) {
                      return JobsController.rowLimitOptions
                          .map(
                            (value) => CheckedPopupMenuItem<int>(
                              value: value,
                              checked: controller.rowLimit == value,
                              child: Text(value == -1 ? "All rows" : "$value rows"),
                            ),
                          )
                          .toList(growable: false);
                    },
                    child: _HeaderChip(
                      icon: Icons.table_rows_rounded,
                      label: controller.rowLimit == -1 ? "Rows: All" : "Rows: ${controller.rowLimit}",
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: JobsTab.values
                    .map(
                      (tab) => ChoiceChip(
                        label: Text(_tabLabel(tab)),
                        selected: controller.tab == tab,
                        onSelected: controller.loading ? null : (_) => controller.setTab(tab),
                      ),
                    )
                    .toList(growable: false),
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: <Widget>[
                  _HeaderChip(
                    icon: controller.workerProcessing ? Icons.play_circle_fill_rounded : Icons.pause_circle_filled_rounded,
                    label: controller.workerProcessing ? "Worker on" : "Worker paused",
                  ),
                  _HeaderChip(
                    icon: Icons.pending_actions_rounded,
                    label: "${controller.processingCount} running · ${controller.queuedCount} queued",
                  ),
                  FilledButton.tonalIcon(
                    onPressed: controller.togglingWorker ? null : onToggleWorker,
                    icon: Icon(controller.workerProcessing ? Icons.pause_rounded : Icons.play_arrow_rounded),
                    label: Text(controller.workerProcessing ? "Pause" : "Resume"),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _HeaderChip extends StatelessWidget {
  const _HeaderChip({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: theme.colorScheme.primary.withValues(alpha: 0.18)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 16, color: theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text(label, style: theme.textTheme.bodySmall),
        ],
      ),
    );
  }
}

class _JobCard extends StatelessWidget {
  const _JobCard({
    required this.tab,
    required this.job,
    required this.isDeleting,
    required this.onOpenChannel,
    required this.onOpenRecording,
    required this.onInspect,
    required this.onDelete,
  });

  final JobsTab tab;
  final DbJob job;
  final bool isDeleting;
  final VoidCallback? onOpenChannel;
  final VoidCallback? onOpenRecording;
  final VoidCallback onInspect;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                _StatusBadge(tab: tab, job: job),
                const SizedBox(width: 8),
                _TaskBadge(label: _formatTask(job.task)),
                const Spacer(),
                Flexible(
                  child: Text(
                    _timeLabel(tab, job),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall,
                    textAlign: TextAlign.end,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                const Icon(Icons.hub_rounded, size: 18),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    job.channelName ?? "Unknown channel",
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                if (onOpenChannel != null)
                  IconButton(
                    onPressed: onOpenChannel,
                    icon: const Icon(Icons.grid_view_rounded),
                    visualDensity: VisualDensity.compact,
                    tooltip: "Open channel",
                  ),
                if (onOpenRecording != null)
                  IconButton(
                    onPressed: onOpenRecording,
                    icon: const Icon(Icons.local_movies_rounded),
                    visualDensity: VisualDensity.compact,
                    tooltip: "Open recording",
                  ),
              ],
            ),
            if ((job.filename ?? "").isNotEmpty) ...<Widget>[
              const SizedBox(height: 4),
              Text(job.filename!, maxLines: 1, overflow: TextOverflow.ellipsis),
            ],
            const SizedBox(height: 10),
            if (tab == JobsTab.active) _ActiveJobProgress(job: job) else Text(_detailText(job), maxLines: 2, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.bodySmall),
            const SizedBox(height: 10),
            Row(
              children: <Widget>[
                Expanded(
                  child: Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: <Widget>[
                      _MetaPill(icon: Icons.speed_rounded, label: _priorityLabel(job.priority)),
                      _MetaPill(icon: Icons.schedule_rounded, label: _formatJobDuration(job)),
                      if (tab == JobsTab.active && !(job.active ?? false)) const _MetaPill(icon: Icons.hourglass_bottom_rounded, label: "Waiting"),
                    ],
                  ),
                ),
                if (tab != JobsTab.active)
                  IconButton(
                    onPressed: onInspect,
                    icon: const Icon(Icons.visibility_outlined),
                    tooltip: "Inspect job",
                  ),
                IconButton(
                  onPressed: isDeleting ? null : onDelete,
                  icon: isDeleting ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.delete_outline_rounded),
                  tooltip: "Delete job",
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.tab, required this.job});

  final JobsTab tab;
  final DbJob job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = switch (_statusTone(tab, job)) {
      _StatusTone.running => (theme.colorScheme.primary.withValues(alpha: 0.12), theme.colorScheme.primary),
      _StatusTone.queued => (Colors.orange.withValues(alpha: 0.14), Colors.orange.shade800),
      _StatusTone.done => (Colors.green.withValues(alpha: 0.14), Colors.green.shade800),
      _StatusTone.error => (theme.colorScheme.error.withValues(alpha: 0.12), theme.colorScheme.error),
      _StatusTone.canceled => (theme.colorScheme.onSurface.withValues(alpha: 0.08), theme.colorScheme.onSurfaceVariant),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(color: colors.$1, borderRadius: BorderRadius.circular(999)),
      child: Text(
        _statusLabel(tab, job),
        style: Theme.of(context).textTheme.labelMedium?.copyWith(color: colors.$2, fontWeight: FontWeight.w700),
      ),
    );
  }
}

class _TaskBadge extends StatelessWidget {
  const _TaskBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: theme.colorScheme.secondaryContainer.withValues(alpha: 0.45),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label, style: Theme.of(context).textTheme.labelMedium?.copyWith(fontWeight: FontWeight.w700)),
    );
  }
}

class _ActiveJobProgress extends StatelessWidget {
  const _ActiveJobProgress({required this.job});

  final DbJob job;

  @override
  Widget build(BuildContext context) {
    if (!(job.active ?? false)) {
      return Text(_detailText(job), maxLines: 2, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.bodySmall);
    }

    final progress = _progressValue(job.progress);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Row(
          children: <Widget>[
            Expanded(
              child: ClipRRect(
                borderRadius: BorderRadius.circular(999),
                child: LinearProgressIndicator(value: progress / 100),
              ),
            ),
            const SizedBox(width: 10),
            Text("${progress.toStringAsFixed(0)}%", style: Theme.of(context).textTheme.bodySmall),
          ],
        ),
        const SizedBox(height: 6),
        Text(_detailText(job), maxLines: 2, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}

class _MetaPill extends StatelessWidget {
  const _MetaPill({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 14),
          const SizedBox(width: 5),
          Text(label, style: Theme.of(context).textTheme.labelSmall),
        ],
      ),
    );
  }
}

class _JobsPaginationBar extends StatelessWidget {
  const _JobsPaginationBar({required this.controller});

  final JobsController controller;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 2),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Row(
            children: <Widget>[
              SizedBox(
                width: 84,
                child: _PagerNavButton(
                  label: "Prev",
                  icon: Icons.chevron_left_rounded,
                  trailingIcon: false,
                  enabled: !controller.loading && controller.currentPage > 1,
                  onPressed: controller.previousPage,
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Center(
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        for (final page in controller.visiblePageItems)
                          if (page == null)
                            const Padding(
                              padding: EdgeInsets.symmetric(horizontal: 4),
                              child: Text("..."),
                            )
                          else
                            Padding(
                              padding: const EdgeInsets.symmetric(horizontal: 3),
                              child: _PagerPageButton(
                                label: page.toString(),
                                selected: page == controller.currentPage,
                                enabled: !controller.loading,
                                onPressed: () => controller.goToPage(page),
                              ),
                            ),
                      ],
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: 84,
                child: _PagerNavButton(
                  label: "Next",
                  icon: Icons.chevron_right_rounded,
                  trailingIcon: true,
                  enabled: !controller.loading && controller.currentPage < controller.totalPages,
                  onPressed: controller.nextPage,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            "Page ${controller.currentPage} / ${controller.totalPages}",
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ],
      ),
    );
  }
}

class _PagerNavButton extends StatelessWidget {
  const _PagerNavButton({
    required this.label,
    required this.icon,
    required this.trailingIcon,
    required this.enabled,
    required this.onPressed,
  });

  final String label;
  final IconData icon;
  final bool trailingIcon;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FilledButton.tonal(
      onPressed: enabled ? onPressed : null,
      style: FilledButton.styleFrom(
        minimumSize: const Size(84, 36),
        padding: const EdgeInsets.symmetric(horizontal: 10),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        visualDensity: VisualDensity.compact,
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: trailingIcon
            ? <Widget>[Text(label), const SizedBox(width: 4), Icon(icon, size: 18)]
            : <Widget>[Icon(icon, size: 18), const SizedBox(width: 4), Text(label)],
      ),
    );
  }
}

class _PagerPageButton extends StatelessWidget {
  const _PagerPageButton({
    required this.label,
    required this.selected,
    required this.enabled,
    required this.onPressed,
  });

  final String label;
  final bool selected;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final style = OutlinedButton.styleFrom(
      minimumSize: const Size(40, 36),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      visualDensity: VisualDensity.compact,
    );

    if (selected) {
      return FilledButton.tonal(
        onPressed: enabled ? onPressed : null,
        style: style,
        child: Text(label),
      );
    }

    return OutlinedButton(
      onPressed: enabled ? onPressed : null,
      style: style,
      child: Text(label),
    );
  }
}

class _DetailField extends StatelessWidget {
  const _DetailField({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 160,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(label.toUpperCase(), style: Theme.of(context).textTheme.labelSmall?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text(value),
        ],
      ),
    );
  }
}

class _DetailBlock extends StatelessWidget {
  const _DetailBlock({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(label.toUpperCase(), style: Theme.of(context).textTheme.labelSmall?.copyWith(fontWeight: FontWeight.w700)),
        const SizedBox(height: 4),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerHighest.withValues(alpha: 0.35),
            borderRadius: BorderRadius.circular(10),
          ),
          child: SelectableText(value),
        ),
      ],
    );
  }
}

enum _StatusTone { running, queued, done, error, canceled }

String _tabLabel(JobsTab tab) {
  return switch (tab) {
    JobsTab.active => "Active",
    JobsTab.completed => "Completed",
    JobsTab.error => "Errors",
    JobsTab.canceled => "Canceled",
  };
}

String _formatTask(DbJobTask? task) {
  return switch (task) {
    DbJobTask.previewFrames => "Preview Frames",
    DbJobTask.analyzeFrames => "Analyze Frames",
    DbJobTask.enhanceVideo => "Enhance Video",
    DbJobTask.convert => "Convert",
    DbJobTask.cut => "Cut",
    DbJobTask.merge => "Merge",
    _ => "Unknown",
  };
}

double _progressValue(String? progress) => (double.tryParse(progress ?? "") ?? 0).clamp(0, 100);

String _priorityLabel(DbJobPriority? priority) {
  return switch (priority) {
    DbJobPriority.value1 => "High",
    DbJobPriority.value3 => "Normal",
    DbJobPriority.value5 => "Low",
    _ => "Normal",
  };
}

_StatusTone _statusTone(JobsTab tab, DbJob job) {
  if (tab == JobsTab.active) {
    return (job.active ?? false) ? _StatusTone.running : _StatusTone.queued;
  }
  if (job.status == DbJobStatus.completed) {
    return _StatusTone.done;
  }
  if (job.status == DbJobStatus.error) {
    return _StatusTone.error;
  }
  return _StatusTone.canceled;
}

String _statusLabel(JobsTab tab, DbJob job) {
  if (tab == JobsTab.active) {
    return (job.active ?? false) ? "Running" : "Queued";
  }
  if (job.status == DbJobStatus.completed) {
    return "Done";
  }
  if (job.status == DbJobStatus.error) {
    return "Error";
  }
  return "Canceled";
}

String _detailText(DbJob job) => job.info ?? job.filename ?? job.command ?? "-";

String _timeLabel(JobsTab tab, DbJob job) {
  if (tab == JobsTab.active) {
    return _relativeTime((job.active ?? false) ? (job.startedAt ?? job.createdAt) : job.createdAt);
  }
  return _relativeTime(job.completedAt ?? job.createdAt);
}

String _relativeTime(String? raw) {
  if (raw == null || raw.isEmpty) {
    return "-";
  }
  final parsed = DateTime.tryParse(raw);
  if (parsed == null) {
    return raw;
  }
  return timeago.format(parsed.toLocal(), locale: "en_short");
}

String _formatJobDuration(DbJob job) {
  final durationMs = job.durationMs;
  if (durationMs == null || durationMs <= 0) {
    return "-";
  }
  final duration = Duration(milliseconds: durationMs);
  if (duration.inHours > 0) {
    return "${duration.inHours}h ${duration.inMinutes.remainder(60)}m";
  }
  if (duration.inMinutes > 0) {
    return "${duration.inMinutes}m ${duration.inSeconds.remainder(60)}s";
  }
  return "${duration.inSeconds}s";
}
