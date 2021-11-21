locals {
  stream_period_mins                    = 5
  stream_period_secs                    = 60 * local.stream_period_mins
  stream_scale_up_threshold             = 0.75
  stream_scale_up_evaluation_period     = 25 / local.stream_period_mins
  stream_scale_up_datapoints_required   = 25 / local.stream_period_mins
  stream_scale_down_threshold           = 0.25
  stream_scale_down_evaluation_period   = 300 / local.stream_period_mins
  stream_scale_down_datapoints_required = 285 / local.stream_period_mins
  stream_scale_down_min_iter_age_mins   = 30
}

resource "aws_cloudwatch_metric_alarm" "nemesis_scale_up" {
  alarm_name                = "${var.kinesis_datastream_name}-scale-up"
  comparison_operator       = "GreaterThanOrEqualToThreshold"
  evaluation_periods        = local.stream_scale_up_evaluation_period
  datapoints_to_alarm       = local.stream_scale_up_datapoints_required
  threshold                 = local.stream_scale_up_threshold
  alarm_description         = "Stream throughput has gone above the scale up threshold"
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.kinesis_scaling_sns_topic.arn]

  metric_query {
    id         = "s1"
    label      = "ShardCount"
    expression = data.aws_kinesis_stream.current_stream.open_shards
  }

  metric_query {
    id    = "m1"
    label = "IncomingBytes"
    metric {
      metric_name = "IncomingBytes"
      namespace   = "AWS/Kinesis"
      period      = local.stream_period_secs
      stat        = "Sum"
      dimensions = {
        StreamName = var.kinesis_datastream_name
      }
    }
  }

  metric_query {
    id    = "m2"
    label = "IncomingRecords"
    metric {
      metric_name = "IncomingRecords"
      namespace   = "AWS/Kinesis"
      period      = local.stream_period_secs
      stat        = "Sum"
      dimensions = {
        StreamName = var.kinesis_datastream_name
      }
    }
  }

  metric_query {
    id         = "e1"
    label      = "FillMissingDataPointsWithZeroForIncomingBytes"
    expression = "FILL(m1,0)"
  }

  metric_query {
    id         = "e2"
    label      = "FillMissingDataPointsWithZeroForIncomingRecords"
    expression = "FILL(m2,0)"
  }

  metric_query {
    id         = "e3"
    label      = "IncomingBytesUsageFactor"
    expression = "e1/(1024*1024*60*${local.stream_period_mins}*s1)"
  }

  metric_query {
    id         = "e4"
    label      = "IncomingRecordsUsageFactor"
    expression = "e2/(1000*60*${local.stream_period_mins}*s1)"
  }

  metric_query {
    id          = "e5"
    label       = "MaxIncomingUsageFactor"
    expression  = "MAX([e3,e4])"
    return_data = true
  }

  lifecycle {
    ignore_changes = [
      tags["LastScaledTimestamp"]
    ]
  }

  depends_on = [
    aws_lambda_function.nemesis_scaling_function
  ]
}

resource "aws_cloudwatch_metric_alarm" "nemesis_scale_down" {
  alarm_name                = "${var.kinesis_datastream_name}-scale-down"
  comparison_operator       = "LessThanThreshold"
  evaluation_periods        = local.stream_scale_down_evaluation_period
  datapoints_to_alarm       = local.stream_scale_down_datapoints_required
  threshold                 = data.aws_kinesis_stream.current_stream.open_shards == 1 ? -1 : local.stream_scale_down_threshold
  alarm_description         = "Stream throughput has gone below the scale down threshold"
  insufficient_data_actions = []
  alarm_actions             = [aws_sns_topic.kinesis_scaling_sns_topic.arn]

  metric_query {
    id         = "s1"
    label      = "ShardCount"
    expression = data.aws_kinesis_stream.current_stream.open_shards
  }

  metric_query {
    id         = "s2"
    label      = "IteratorAgeMinutesToBlockScaledowns"
    expression = local.stream_scale_down_min_iter_age_mins
  }

  metric_query {
    id    = "m1"
    label = "IncomingBytes"
    metric {
      metric_name = "IncomingBytes"
      namespace   = "AWS/Kinesis"
      period      = local.stream_period_secs
      stat        = "Sum"
      dimensions = {
        StreamName = var.kinesis_datastream_name
      }
    }
  }

  metric_query {
    id    = "m2"
    label = "IncomingRecords"
    metric {
      metric_name = "IncomingRecords"
      namespace   = "AWS/Kinesis"
      period      = local.stream_period_secs
      stat        = "Sum"
      dimensions = {
        StreamName = var.kinesis_datastream_name
      }
    }
  }

  metric_query {
    id    = "m3"
    label = "GetRecords.IteratorAgeMilliseconds"
    metric {
      metric_name = "GetRecords.IteratorAgeMilliseconds"
      namespace   = "AWS/Kinesis"
      period      = local.stream_period_secs
      stat        = "Maximum"
      dimensions = {
        StreamName = var.kinesis_datastream_name
      }
    }
  }

  metric_query {
    id         = "e1"
    label      = "FillMissingDataPointsWithZeroForIncomingBytes"
    expression = "FILL(m1,0)"
  }

  metric_query {
    id         = "e2"
    label      = "FillMissingDataPointsWithZeroForIncomingRecords"
    expression = "FILL(m2,0)"
  }

  metric_query {
    id         = "e3"
    label      = "IncomingBytesUsageFactor"
    expression = "e1/(1024*1024*60*${local.stream_period_mins}*s1)"
  }

  metric_query {
    id         = "e4"
    label      = "IncomingRecordsUsageFactor"
    expression = "e2/(1000*60*${local.stream_period_mins}*s1)"
  }

  metric_query {
    id         = "e5"
    label      = "IteratorAgeAdjustedFactor"
    expression = "(FILL(m3,0)/1000/60)*(${local.stream_scale_down_threshold}/s2)"
  }

  metric_query {
    id          = "e6"
    label       = "MaxIncomingUsageFactor"
    expression  = "MAX([e3,e4,e5])"
    return_data = true
  }

  lifecycle {
    ignore_changes = [
      tags["LastScaledTimestamp"]
    ]
  }

  depends_on = [
    aws_lambda_function.nemesis_scaling_function
  ]
}