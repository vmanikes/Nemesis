// Package kinesis contains the methods to integrate with kinesis
package kinesis

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/vmanikes/Nemesis/logging"
	"go.uber.org/zap"
)

type Client struct {
	kinesisClient *kinesis.Client
}

// New creates a new kinesis client and returns if successfully initialized
func New(ctx context.Context) (*Client, error) {
	logger := logging.WithContext(ctx)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("unable to load the default config for aws")
		return nil, err
	}

	return &Client{
		kinesisClient: kinesis.NewFromConfig(cfg),
	}, nil
}

// GetShardCount takes in a kinesis stream name and returns the shard count for the stream
func (c *Client) GetShardCount(ctx context.Context, streamName string) (int, error) {
	logger := logging.WithContext(ctx)

	summary, err := c.kinesisClient.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
		StreamName: &streamName,
	})
	if err != nil {
		logger.Error("unable describe stream summary",
			zap.String("stream-name", streamName))
		return 0, err
	}

	return int(*summary.StreamDescriptionSummary.OpenShardCount), nil
}

// UpdateShardCount takes in a stream name and shard count and updates the kinesis stream
func (c *Client) UpdateShardCount(ctx context.Context, streamName string, shardCount int32) error {
	logger := logging.WithContext(ctx)
	
	_, err := c.kinesisClient.UpdateShardCount(ctx, &kinesis.UpdateShardCountInput{
		ScalingType:      types.ScalingTypeUniformScaling,
		StreamName:       &streamName,
		TargetShardCount: aws.Int32(shardCount),
	})
	if err != nil {
		logger.Error("unable update shard count",
			zap.String("stream-name", streamName),
			zap.Int32("new-shard-count", shardCount))
		return err
	}

	return nil
}