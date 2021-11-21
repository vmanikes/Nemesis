# Nemesis
[![Go Report Card](https://goreportcard.com/badge/github.com/vmanikes/Nemesis)](https://goreportcard.com/report/github.com/vmanikes/Nemesis)

TODO:
- DOC: Make sure you have this to your stream
  ```
  shard_level_metrics = [
    "IncomingBytes",
  ]

  lifecycle {
    ignore_changes = [
      shard_count, # Kinesis autoscaling will change the shard count outside of terraform
    ]
  }
  ```