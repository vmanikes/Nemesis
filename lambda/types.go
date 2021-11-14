package main

import (
	"context"
	"errors"
	"github.com/vmanikes/Nemesis/logging"
	"go.uber.org/zap"
)

type AlarmInformation map[string]interface{}

// GetAlarmName extracts the alarm name from the event payload
func (a AlarmInformation) GetAlarmName(ctx context.Context) (string, error) {
	logger := logging.WithContext(ctx)

	alarmName, ok := a["AlarmName"].(string)
	if !ok {
		err := errors.New("non string alarm name")
		logger.Error(err.Error(),
			zap.Any("alarm-name", a["AlarmName"]))
		return "", err
	}

	return alarmName, nil
}

// GetAlarmArn extracts the alarm arn from the event payload
func (a AlarmInformation) GetAlarmArn(ctx context.Context) (string, error) {
	logger := logging.WithContext(ctx)

	alarmArn, ok := a["AlarmName"].(string)
	if !ok {
		err := errors.New("non string alarm arn")
		logger.Error(err.Error(),
			zap.Any("alarm-name", a["AlarmName"]))
		return "", err
	}

	return alarmArn, nil
}

// GetStateChangeTime extracts the state change time from event payload
func (a AlarmInformation) GetStateChangeTime(ctx context.Context) (string, error) {
	logger := logging.WithContext(ctx)

	stateChangeTime, ok := a["StateChangeTime"].(string)
	if !ok {
		err := errors.New("non string state change time")
		logger.Error(err.Error(),
			zap.Any("state-change-time", a["StateChangeTime"]))
		return "", err
	}

	return stateChangeTime, nil
}

// GetStreamName extracts the stream name from the event payload
func (a AlarmInformation) GetStreamName() (stream string) {
	for _, metric := range a["Trigger"].(map[string]interface{})["Metrics"].([]interface{}) {
		if metric.(map[string]interface{})["MetricStat"] != nil {
			if metric.(map[string]interface{})["Id"] == "m1" || metric.(map[string]interface{})["Id"] == "m2" {
				for _, dimension := range metric.(map[string]interface{})["MetricStat"].(map[string]interface{})["Metric"].(map[string]interface{})["Dimensions"].([]interface{}) {
					stream = dimension.(map[string]interface{})["value"].(string)
				}
				break
			}
		}
	}

	return stream
}