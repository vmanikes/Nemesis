data "aws_iam_policy_document" "nemesis_scaling_lambda_trust_policy_document" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "nemesis_scaling_lambda_role" {
  name               = "Nemesis-${var.kinesis_datastream_name}-role"
  assume_role_policy = data.aws_iam_policy_document.nemesis_scaling_lambda_trust_policy_document.json
  tags               = var.tags
}

data "aws_iam_policy_document" "nemesis_scaling_lambda_policy_document" {
  statement {
    sid       = "AllowCreateCloudWatchAlarms"
    effect    = "Allow"
    resources = ["*"]

    actions = [
      "cloudwatch:DescribeAlarms",
      "cloudwatch:GetMetricData",
      "cloudwatch:ListMetrics",
      "cloudwatch:PutMetricAlarm",
      "cloudwatch:PutMetricData",
      "cloudwatch:ListTagsForResource",
      "cloudwatch:SetAlarmState",
      "cloudwatch:TagResource"
    ]
  }

  statement {
    sid       = "AllowLoggingToCloudWatch"
    effect    = "Allow"
    resources = ["*"]

    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
  }

  statement {
    sid       = "AllowReadFromKinesis"
    effect    = "Allow"
    resources = ["arn:aws:kinesis:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:stream/*"]

    actions = [
      "kinesis:DescribeStreamSummary",
      "kinesis:AddTagsToStream",
      "kinesis:ListTagsForStream",
      "kinesis:UpdateShardCount",
    ]
  }

  statement {
    sid       = "AllowPublishToSNS"
    effect    = "Allow"
    resources = ["arn:aws:sns:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:*"]

    actions = [
      "sns:Publish",
    ]
  }

  statement {
    sid       = "AllowChangeFunctionConcurrencyForLambda"
    effect    = "Allow"
    resources = ["arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:*"]

    actions = [
      "lambda:PutFunctionConcurrency",
      "lambda:DeleteFunctionConcurrency"
    ]
  }
}

resource "aws_iam_policy" "nemesis_scaling_lambda_policy" {
  name        = "Nemesis-${var.kinesis_datastream_name}-policy"
  path        = "/"
  description = "Policy for Central Logging Kinesis Auto-Scaling Lambda"
  policy      = data.aws_iam_policy_document.nemesis_scaling_lambda_policy_document.json
}

resource "aws_iam_role_policy_attachment" "attach_kinesis_scaling_lambda_policy" {
  role       = aws_iam_role.nemesis_scaling_lambda_role.name
  policy_arn = aws_iam_policy.nemesis_scaling_lambda_policy.arn
}