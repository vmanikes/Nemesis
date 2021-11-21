// Package cloudwatch contains the methods to integrate with cloudwatch
package cloudwatch

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	kinesis "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/vmanikes/Nemesis/constants"
	"github.com/vmanikes/Nemesis/logging"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type Client struct {
	cloudwatchClient *cloudwatch.Client
}

// New creates and initialized the cloudwatch client
func New(ctx context.Context) (*Client, error) {
	logger := logging.WithContext(ctx)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("unable to load the default config for aws")
		return nil, err
	}

	return &Client{
		cloudwatchClient: cloudwatch.NewFromConfig(cfg),
	}, nil
}

// GetAlarmNames takes in the triggered alarm name and ARN. It returns the scale up and scale down alarm names along with
// the action
func (c *Client) GetAlarmNames(ctx context.Context, currentAlarmName, currentAlarmArn string) (
	scaleUpAlarmName, scaleDownAlarmName, currentAction, lastAlarmActionTimestamp string, err error){

	logger := logging.WithContext(ctx)
	response, err := c.cloudwatchClient.ListTagsForResource(ctx, &cloudwatch.ListTagsForResourceInput{
		ResourceARN: aws.String(currentAlarmArn),
	})
	if err != nil {
		logger.Error("unable to list tags for resource",
			zap.String("alarm-name", currentAlarmName),
			zap.String("alarm-arn", currentAlarmArn))
		return "", "", "", "", err
	}

	var (
		scaleDownSuffix = "-scale-down"
		scaleUpSuffix = "-scale-up"
	)

	if strings.HasSuffix(currentAlarmName, scaleUpSuffix) {
		currentAction = "Up"
		scaleUpAlarmName = currentAlarmName
		scaleDownAlarmName = currentAlarmName[0:len(currentAlarmName)-len(scaleUpSuffix)] + scaleDownSuffix
	} else if strings.HasSuffix(currentAlarmName, scaleDownSuffix) {
		currentAction = "Down"
		scaleUpAlarmName = currentAlarmName[0:len(currentAlarmName)-len(scaleDownSuffix)] + scaleUpSuffix
		scaleDownAlarmName = currentAlarmName
	}

	for _, tag := range response.Tags {
		if aws.ToString(tag.Key) == "LastScaledTimestamp" {
			lastAlarmActionTimestamp = aws.ToString(tag.Value)
		}
	}

	return scaleUpAlarmName, scaleDownAlarmName, currentAction, lastAlarmActionTimestamp, nil
}

// SetAlarmState takes alarm name, state and reason and changes the state of the alarm
func (c *Client) SetAlarmState(ctx context.Context, alarmName, state, reason string) error {
	logger := logging.WithContext(ctx)

	_, err := c.cloudwatchClient.SetAlarmState(ctx, &cloudwatch.SetAlarmStateInput{
		AlarmName:       aws.String(alarmName),
		StateReason:     &reason,
		StateValue: types.StateValue(state),
	})
	if err != nil {
		logger.Error("unable to set alarm state",
			zap.Error(err))
		return err
	}

	return nil
}

// UpdateAlarm updates the alarm metrics with the new shard count
func (c *Client) UpdateAlarm(ctx context.Context, alarmName, streamName, snsARN string, isScaleDown bool, shardCount int) error {
	logger := logging.WithContext(ctx)

	input := &cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String(alarmName),
		AlarmDescription:   aws.String("Alarm to scale Kinesis stream"),
		ActionsEnabled:     aws.Bool(true),
		AlarmActions:       []string{snsARN},
		TreatMissingData:   aws.String("ignore"),
	}

	metrics := make([]types.MetricDataQuery, 0)

	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("m1"),
		Label:      aws.String(string(kinesis.MetricsNameIncomingBytes)),
		MetricStat: &types.MetricStat{
			Metric: &types.Metric{
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("StreamName"),
						Value: aws.String(streamName),
					},
				},
				MetricName: aws.String(string(kinesis.MetricsNameIncomingBytes)),
				Namespace:  aws.String("AWS/Kinesis"),
			},
			Period: aws.Int32(int32(60 * constants.ScalePeriodMinutes)),
			Stat:   aws.String(string(types.StatisticSum)),
		},
		ReturnData: aws.Bool(false),
	})

	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("m2"),
		Label:      aws.String(string(kinesis.MetricsNameIncomingRecords)),
		MetricStat: &types.MetricStat{
			Metric: &types.Metric{
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("StreamName"),
						Value: aws.String(streamName),
					},
				},
				MetricName: aws.String(string(kinesis.MetricsNameIncomingRecords)),
				Namespace:  aws.String("AWS/Kinesis"),
			},
			Period: aws.Int32(int32(60 * constants.ScalePeriodMinutes)),
			Stat:   aws.String(string(types.StatisticSum)),
		},
		ReturnData: aws.Bool(false),
	})

	if isScaleDown {
		input.Threshold = aws.Float64(constants.ScaleDownThreshold)
		input.DatapointsToAlarm = aws.Int32(int32(constants.DataPointsToScaleDown))
		input.EvaluationPeriods = aws.Int32(int32(constants.ScaleDownEvaluationPeriodMinutes))
		input.ComparisonOperator = types.ComparisonOperatorLessThanThreshold

		metrics = append(metrics, types.MetricDataQuery{
			Id:         aws.String("m3"),
			Label:      aws.String("GetRecords.IteratorAgeMilliseconds"),
			ReturnData: aws.Bool(false),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String("AWS/Kinesis"),
					MetricName: aws.String("GetRecords.IteratorAgeMilliseconds"),
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("StreamName"),
							Value: aws.String(streamName),
						},
					},
				},
				Period: aws.Int32(int32(60 * constants.ScalePeriodMinutes)),
				Stat:   aws.String(string(types.StatisticMaximum)),
			},
		})

		metrics = append(metrics, types.MetricDataQuery{
			Id:         aws.String("e5"),
			Expression: aws.String(fmt.Sprintf("(FILL(m3,0)/1000/60)*(%0.5f/s2)", constants.ScaleDownThreshold)),
			Label:      aws.String("IteratorAgeAdjustedFactor"),
			ReturnData: aws.Bool(false),
		})

		metrics = append(metrics, types.MetricDataQuery{
			Id:         aws.String("e6"),
			Expression: aws.String("MAX([e3,e4,e5])"),
			Label:      aws.String("MaxIncomingUsageFactor"),
			ReturnData: aws.Bool(true),
		})

		metrics = append(metrics, types.MetricDataQuery{
			Id:         aws.String("s2"),
			Expression: aws.String(fmt.Sprintf("%d", constants.ScaleDownMinIterAgeMinutes)),
			Label:      aws.String("IteratorAgeMinutesToBlockScaleDowns"),
			ReturnData: aws.Bool(false),
		})

	} else {
		input.Threshold = aws.Float64(constants.ScaleUpThreshold)
		input.DatapointsToAlarm = aws.Int32(int32(constants.DataPointsToScaleUp))
		input.EvaluationPeriods = aws.Int32(int32(constants.ScaleUpEvaluationPeriodMinutes))
		input.ComparisonOperator = types.ComparisonOperatorGreaterThanOrEqualToThreshold

		metrics = append(metrics, types.MetricDataQuery{
			Id:         aws.String("e6"),
			Expression: aws.String("MAX([e3,e4])"), // Scale up doesn't look at iterator age, only bytes/sec, records/sec
			Label:      aws.String("MaxIncomingUsageFactor"),
			ReturnData: aws.Bool(true),
		})
	}

	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("e1"),
		Expression: aws.String("FILL(m1,0)"),
		Label:      aws.String("FillMissingDataPointsWithZeroForIncomingBytes"),
		ReturnData: aws.Bool(false),
	})
	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("e2"),
		Expression: aws.String("FILL(m2,0)"),
		Label:      aws.String("FillMissingDataPointsWithZeroForIncomingRecords"),
		ReturnData: aws.Bool(false),
	})
	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("e3"),
		Expression: aws.String(fmt.Sprintf("e1/(1024*1024*60*%d*s1)", constants.ScalePeriodMinutes)),
		Label:      aws.String("IncomingBytesUsageFactor"),
		ReturnData: aws.Bool(false),
	})
	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("e4"),
		Expression: aws.String(fmt.Sprintf("e2/(1000*60*%d*s1)", constants.ScalePeriodMinutes)),
		Label:      aws.String("IncomingRecordsUsageFactor"),
		ReturnData: aws.Bool(false),
	})

	shardCountStr := strconv.Itoa(shardCount)
	metrics = append(metrics, types.MetricDataQuery{
		Id:         aws.String("s1"),
		Expression: aws.String(shardCountStr),
		Label:      aws.String("ShardCount"),
		ReturnData: aws.Bool(false),
	})

	input.Metrics = metrics

	_, err := c.cloudwatchClient.PutMetricAlarm(ctx, input)
	if err != nil {
		logger.Error("unable to update alarm",
			zap.Error(err))
		return err
	}

	return nil
}

// GetAlarmArns takes in the scale up and scale dpwn alarm names and returns their arns
func (c *Client) GetAlarmArns(ctx context.Context, scaleUpAlarmName, scaleDownAlarmName string) (string, string, error) {
	logger := logging.WithContext(ctx)

	describeAlarmsResponse, err := c.cloudwatchClient.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{scaleUpAlarmName, scaleDownAlarmName},
	})
	if err != nil {
		logger.Error("unable to describe alarms",
			zap.Error(err))
		return "", "", err
	}

	var (
		scaleUpAlarmArn string
		scaleDownAlarmArn string
	)

	for _, alarm := range describeAlarmsResponse.MetricAlarms {
		if *alarm.AlarmName == scaleUpAlarmName {
			scaleUpAlarmArn = *alarm.AlarmArn
		}
		if *alarm.AlarmName == scaleDownAlarmName {
			scaleDownAlarmArn = *alarm.AlarmArn
		}
	}

	return scaleUpAlarmArn, scaleDownAlarmArn, nil
}

// TagAlarm tags the alarm with the scale action, complementary alarm and adds the last scale timestamp to the tag
func (c *Client) TagAlarm(ctx context.Context, alarmArn, actionValue, alarmName, lastScaleTimestamp string) error {
	logger := logging.WithContext(ctx)

	_, err := c.cloudwatchClient.TagResource(ctx, &cloudwatch.TagResourceInput{
		ResourceARN: &alarmArn,
		Tags: []types.Tag{
			{
				Key:   aws.String("ScaleAction"),
				Value: &actionValue,
			},
			{
				Key:   aws.String("ComplimentaryAlarm"),
				Value: &alarmName,
			},
			{
				Key:   aws.String("LastScaledTimestamp"),
				Value: &lastScaleTimestamp,
			},
		},
	})
	if err != nil {
		logger.Error("unable to tag alarm",
			zap.Error(err))
		return err
	}

	return nil
}