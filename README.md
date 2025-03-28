# replay

## Purpose

This CLI tool helps engineers review and manage dead-lettered messages across multiple message brokers. 
Users can iterate through each dead-lettered message and choose to discard or reprocess it by moving it to a different queue.

## Supported Message Brokers

- GCP Pub/Sub

## Usage Overview

1. Install the CLI.
2. Run the tool to review or move dead-lettered messages.
3. Select specific messages to discard or reprocess.

## Usage Examples

### Move Operation

To move messages from a GCP Pub/Sub subscription to a GCP Pub/Sub topic, run:

```
replay move \
  --source-type GCP_PUBSUB_SUBSCRIPTION \
  --destination-type GCP_PUBSUB_TOPIC \
  --source projects/[project]/subscriptions/[name] \
  --destination projects/[project]/topics/[name]
```

By default, the command moves all messages from the source to the destination until there are no more messages in the source. Each message is moved one at a time by following this process:
- Poll 1 message from the source
- Publish the message to the destination
- Acknowledge the message at the source

To move only a certain number of messages, add the --count [integer] argument.

### Dead Letter Review

To review and process dead-lettered messages, run:

```
replay dlr \
  --source-type GCP_PUBSUB_SUBSCRIPTION \
  --destination-type GCP_PUBSUB_TOPIC \
  --source projects/[project]/subscriptions/[name] \
  --destination projects/[project]/topics/[name]
```

- Use the --pretty-json flag to display message data as formatted JSON.

## Full CLI Usage Documentation

[Click here](./docs/replay.md) to view the full CLI usage documentation.

## Versioning

This project follows Semantic Versioning 2.0 guidelines: 
- MAJOR version updates for incompatible changes
- MINOR version updates for added functionality
- PATCH version updates for bug fixes
