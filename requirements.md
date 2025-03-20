# 1. Product Requirements

- I. Project must be a command line interface tool
- II. User must be able to move messages from a source to a destination
- III. User must be able to review dead-lettered messages from a source and choose whether to discard or move the message
- IV. Supported message sources
  - 1. GCP PubSub subscription
- V. Supported message destinations
  - 1. GCP PubSub topic
- VI. Supported authentication information sources
  - 1. The user-level authentication information used and managed by the official GCloud CLI tool

# 2. Technical Requirements

- I. Project must be written in GoLang
- II. Project must use the Cobra + Viper tech stack for building a GoLang CLI
