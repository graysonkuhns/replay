# replay

## Purpose

This CLI tool helps engineers review and manage dead-lettered messages across multiple message brokers. 
Users can iterate through each dead-lettered message and choose to discard or reprocess it by moving it to a different queue.

## Usage Overview

1. Install the CLI.
2. Run the tool to review or move dead-lettered messages.
3. Select specific messages to discard or reprocess.

## Supported Message Brokers

- GCP Pub/Sub (initial implementation)
- AWS SNS+SQS (planned)

## Versioning

This project follows Semantic Versioning 2.0 guidelines: 
- MAJOR version updates for incompatible changes
- MINOR version updates for added functionality
- PATCH version updates for bug fixes
