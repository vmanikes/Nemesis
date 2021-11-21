resource "aws_sns_topic" "nemesis_scaling_sns_topic" {
  name = "Nemesis-${var.kinesis_datastream_name}-scaling-topic"
}

resource "aws_sns_topic_subscription" "nemesis_scaling_sns_topic_subscription" {
  topic_arn = aws_sns_topic.nemesis_scaling_sns_topic.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.nemesis_scaling_function.arn
}

resource "aws_lambda_permission" "nemesis_scaling_sns_topic_permission" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.nemesis_scaling_function.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.nemesis_scaling_sns_topic.arn
}