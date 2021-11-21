data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_kinesis_stream" "current_stream" {
  name = var.kinesis_datastream_name
}