#!/usr/bin/env bash

gcloud pubsub topics list --format="value(name)" | xargs -I {} gcloud pubsub topics delete {}
