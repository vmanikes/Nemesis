resource "aws_cloudwatch_metric_alarm" "nemesis_scaling_fatal_errors" {
  alarm_name                = "${aws_lambda_function.nemesis_scaling_function.function_name}-fatal-errors"
  comparison_operator       = "GreaterThanThreshold"
  evaluation_periods        = "1"
  metric_name               = "FATAL_ERROR_KINESIS_SCALING"
  namespace                 = "AWS/Lambda"
  period                    = "60"
  statistic                 = "Average"
  threshold                 = "0"
  alarm_description         = "This metric monitors fatal errors in the kinesis scaling lambda"
  insufficient_data_actions = []

  dimensions = {
    FunctionName = aws_lambda_function.nemesis_scaling_function.function_name
  }
}

resource "aws_lambda_function_event_invoke_config" "nemesis_scaling_function_async_config" {
  function_name          = aws_lambda_function.nemesis_scaling_function.function_name
  maximum_retry_attempts = 0 # We do not want to retry on failure as the next alarm trigger will pick it up
}

resource "aws_lambda_function" "nemesis_scaling_function" {
  filename                       = data.archive_file.nemesis_scaling_function_zip.output_path
  function_name                  = "Nemesis-${var.kinesis_datastream_name}-scaling-function"
  handler                        = "main"
  role                           = aws_iam_role.kinesis_scaling_lambda_role.arn
  runtime                        = "go1.x"
  source_code_hash               = data.archive_file.nemesis_scaling_function_zip.output_base64sha256
  timeout                        = 900
  memory_size                    = 512
  reserved_concurrent_executions = 1
}

// TODO Check if zip can be part of terraform module
data "archive_file" "nemesis_scaling_function_zip" {
  type        = "zip"
  source_file = "../main"
  output_path = "../nemesis_${var.kinesis_datastream_name}_scaling.zip"
}
