#!/usr/bin/env bash

gcloud pubsub subscriptions list --format="value(name)" | xargs -I {} gcloud pubsub subscriptions delete {}

