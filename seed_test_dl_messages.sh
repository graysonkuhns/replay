#!/usr/bin/env bash

ENVIRONMENT=${ENVIRONMENT:-default}
TOPIC_NAME="${ENVIRONMENT}-events-dead-letter"
for i in {1..5}; do
    MESSAGE="{\"message\": \"Test DL message $i\"}"
    echo "Publishing message $i to topic ${TOPIC_NAME}"
    gcloud pubsub topics publish "$TOPIC_NAME" --message "$MESSAGE"
done
