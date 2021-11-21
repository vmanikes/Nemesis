package types

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

var (
	testAlarmInfo AlarmInformation
	badAlarmInfo AlarmInformation
)

func TestMain(m *testing.M) {
	fileBytes, err := ioutil.ReadFile("../tests/alarm1.json")
	if err != nil {
		log.Println("Unable to read the alarm file: ", err)
		os.Exit(1)
	}

	err = json.Unmarshal(fileBytes, &testAlarmInfo)
	if err != nil {
		log.Println("Unable unmarshal alarm file: ", err)
		os.Exit(1)
	}

	fileBytes, err = ioutil.ReadFile("../tests/bad_alarm.json")
	if err != nil {
		log.Println("Unable to read the alarm file: ", err)
		os.Exit(1)
	}

	err = json.Unmarshal(fileBytes, &badAlarmInfo)
	if err != nil {
		log.Println("Unable unmarshal alarm file: ", err)
		os.Exit(1)
	}

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestAlarmInformation_GetAlarmName(t *testing.T) {
	alarmName, err := testAlarmInfo.GetAlarmName(context.Background())
	if err != nil {
		t.Error("unable to get alarm name: ", err)
		return
	}

	assert.Equal(t, "alarm-scale-up", alarmName)
}

func TestAlarmInformation_GetAlarmName_Error(t *testing.T) {
	_, err := badAlarmInfo.GetAlarmName(context.Background())
	assert.Error(t, err)
}

func TestAlarmInformation_GetAlarmArn(t *testing.T) {
	alarmArn, err := testAlarmInfo.GetAlarmArn(context.Background())
	if err != nil {
		t.Error("unable to get alarm arn: ", err)
		return
	}

	assert.Equal(t, "arn:aws:cloudwatch:us-east-1:321434131231:alarm:alarm-scale-up", alarmArn)
}

func TestAlarmInformation_GetAlarmArn_Error(t *testing.T) {
	_, err := badAlarmInfo.GetAlarmArn(context.Background())
	assert.Error(t, err)
}

func TestAlarmInformation_GetStateChangeTime(t *testing.T) {
	stateChangeTime, err := testAlarmInfo.GetStateChangeTime(context.Background())
	if err != nil {
		t.Error("unable to get state change time: ", err)
		return
	}

	assert.Equal(t, "2020-04-23T21:17:44.775+0000", stateChangeTime)
}

func TestAlarmInformation_GetStateChangeTime_Error(t *testing.T) {
	_, err := badAlarmInfo.GetStateChangeTime(context.Background())
	assert.Error(t, err)
}

func TestAlarmInformation_GetStreamNames(t *testing.T) {
	assert.Equal(t, "test-stream", testAlarmInfo.GetStreamName())
}