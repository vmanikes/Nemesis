package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/vmanikes/Nemesis/cloudwatch"
	"github.com/vmanikes/Nemesis/constants"
	"github.com/vmanikes/Nemesis/kinesis"
	"github.com/vmanikes/Nemesis/logging"
	"go.uber.org/zap"
	"time"
)

func handleRequest(ctx context.Context, snsEvent events.SNSEvent) {
	logger := logging.WithContext(ctx)

	if len(snsEvent.Records) == 0 {
		logger.Error("SNS event does not contain any records")
		return
	}

	snsRecord := snsEvent.Records[0].SNS

	var alarmInformation AlarmInformation

	err := json.Unmarshal([]byte(snsRecord.Message), &alarmInformation)
	if err != nil {
		logger.Error("unable to unmarshal alarm information from SNS",
			zap.Error(err))
		return
	}

	alarmName, err := alarmInformation.GetAlarmName(ctx)
	if err != nil {
		return
	}

	alarmArn, err := alarmInformation.GetAlarmArn(ctx)
	if err != nil {
		return
	}

	ctx = logging.NewContext(ctx, zap.String("alarm-name", alarmName))

	cloudwatchClient, err := cloudwatch.New(ctx)
	if err != nil {
		return
	}

	scaleUpAlarmName, scaleDownAlarmName, currentAction, lastAlarmActionTimestamp, err := cloudwatchClient.GetAlarmNames(ctx, alarmName, alarmArn)
	if err != nil {
		return
	}

	ctx = logging.NewContext(ctx,
		zap.String("scale-up-alarm", scaleUpAlarmName),
		zap.String("scale-down-alarm", scaleDownAlarmName),
		zap.String("last-alarm-action", lastAlarmActionTimestamp))

	if currentAction == "" {
		logger.Error("current scale action is empty")
		return
	}

	ctx = logging.NewContext(ctx, zap.String("scale-action", currentAction))

	streamName := alarmInformation.GetStreamName()
	ctx = logging.NewContext(ctx, zap.String("stream-name", streamName))

	stateChangeTime, err := alarmInformation.GetStateChangeTime(ctx)
	if err != nil {
		return
	}

	if !ShouldScaleKinesis(lastAlarmActionTimestamp, stateChangeTime) {
		reason := "Scale-" + currentAction + " event rejected. Changing alarm state back to Insufficient Data."
		_ = cloudwatchClient.SetAlarmState(ctx, alarmName, string(types.StateValueInsufficientData), reason)
		return
	}

	kinesisClient, err := kinesis.New(ctx)
	if err != nil {
		return
	}

	shardCount, err := kinesisClient.GetShardCount(ctx, streamName)
	if err != nil {
		return
	}

	newShardCount := CalculateShardCount(currentAction, shardCount)

	err = kinesisClient.UpdateShardCount(ctx, streamName, int32(newShardCount))
	if err != nil {
		return
	}

	alarmLastScaledTimestampValue := time.Now().Format("2006-01-02T15:04:05.000+0000")

	err = cloudwatchClient.UpdateAlarm(ctx, scaleUpAlarmName, streamName, alarmArn, false, newShardCount)
	if err != nil {
		return
	}

	err = cloudwatchClient.SetAlarmState(ctx, scaleUpAlarmName, string(types.StateValueInsufficientData), "Metric math and threshold value update")
	if err != nil {
		return
	}

	err = cloudwatchClient.UpdateAlarm(ctx, scaleDownAlarmName, streamName, alarmArn, true, newShardCount)
	if err != nil {
		return
	}

	err = cloudwatchClient.SetAlarmState(ctx, scaleDownAlarmName, string(types.StateValueInsufficientData), "Metric math and threshold value update")
	if err != nil {
		return
	}

	scaleUpAlarmArn, scaleDownAlarmArn, err := cloudwatchClient.GetAlarmArns(ctx, scaleUpAlarmName, scaleDownAlarmName)
	if err != nil {
		return
	}

	err = cloudwatchClient.TagAlarm(ctx, scaleUpAlarmArn, "Up", scaleDownAlarmName, alarmLastScaledTimestampValue)
	if err != nil {
		return
	}

	err = cloudwatchClient.TagAlarm(ctx, scaleDownAlarmArn, "Down", scaleUpAlarmName, alarmLastScaledTimestampValue)
	if err != nil {
		return
	}
}

// CalculateShardCount returns the new shard count based on the scaling action and the updates scale down threshold
// the down threshold will be -1.0 with the new calculation turns out to be 1
func CalculateShardCount(scaleAction string, currentShardCount int) int {
	var targetShardCount int

	if scaleAction == "Up" {
		targetShardCount = currentShardCount * 2
	}

	if scaleAction == "Down" {
		targetShardCount = currentShardCount / 2
		// Set to minimum shard count
		if targetShardCount <= 1 {
			targetShardCount = 1
			// At minimum shard count,set the scale down threshold to -1, so that scale down alarm remains in OK state
			constants.ScaleDownThreshold = -1.0
		}
	}

	return targetShardCount
}

// ShouldScaleKinesis checks if the kinesis stream should be scaled or not. This is just to avoid a race condition on
// scaling kinesis like crazy
func ShouldScaleKinesis(lastScaledTimestamp, alarmTime string) bool {
	var (
		firstEverScaleAttempt = true
	)

	if lastScaledTimestamp == "" {
		firstEverScaleAttempt = true
	} else {
		firstEverScaleAttempt = false
	}

	if firstEverScaleAttempt {
		return true
	}

	var stateChangeTime, stateChangeParseErr = time.Parse("2006-01-02T15:04:05.000+0000", alarmTime)
	var lastScaled, lastScaledTimestampParseErr = time.Parse("2006-01-02T15:04:05.000+0000", lastScaledTimestamp)

	if lastScaledTimestampParseErr != nil || stateChangeParseErr != nil {
		return true
	}

	if stateChangeTime.Before(lastScaled) || stateChangeTime.Equal(lastScaled) {
		return false
	}

	// Too soon since the last scaling event
	var nextAllowedScalingEvent = lastScaled.Add(time.Minute * time.Duration(constants.ScalePeriodMinutes))
	if stateChangeTime.Before(nextAllowedScalingEvent) {
		return false
	}

	return true
}

func main()  {
	lambda.Start(handleRequest)
}