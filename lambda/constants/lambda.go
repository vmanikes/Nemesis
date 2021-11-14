// Package constants contains Lambda specific constants and default vars
package constants

var (
	// ScalePeriodMinutes specifies the default scaling period in minutes
	ScalePeriodMinutes int64 = 5
	// ScaleUpEvaluationPeriodMinutes specifies the evaluation period for scaling up streams. Default is 25 minutes
	ScaleUpEvaluationPeriodMinutes = 25 / ScalePeriodMinutes
	// ScaleDownEvaluationPeriodMinutes specifies the evaluation period for scaling down streams. Default is 300 minutes
	ScaleDownEvaluationPeriodMinutes = 300 / ScalePeriodMinutes
	// DataPointsToScaleUp specifies the number of data points to scale up
	DataPointsToScaleUp              = 25 / ScalePeriodMinutes
	// DataPointsToScaleDown specifies the number of data points to scale down
	DataPointsToScaleDown            = 285 / ScalePeriodMinutes
	// ScaleDownMinIterAgeMinutes Will wait for the lambdas/shards to clear backlog
	ScaleDownMinIterAgeMinutes int64 = 30
	// ScaleUpThreshold sets the upper limit at crossing which the shards will scale up
	ScaleUpThreshold                 = 0.25
	// ScaleDownThreshold sets the lower limit at crossing which the shards will scale down
	ScaleDownThreshold               = 0.075
)
