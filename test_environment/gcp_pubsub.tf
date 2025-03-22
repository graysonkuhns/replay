resource "google_pubsub_topic" "events" {
  name   = "${local.environment}-events"
  labels = {
    environment = local.environment
  }
}

resource "google_pubsub_topic" "events_dead_letter" {
  name   = "${local.environment}-events-dead-letter"
  labels = {
    environment = local.environment
  }
}

resource "google_pubsub_subscription" "events" {
  name  = "${local.environment}-events-subscription"
  topic = google_pubsub_topic.events.name
  labels = {
    environment = local.environment
  }
}

resource "google_pubsub_subscription" "events_dead_letter" {
  name  = "${local.environment}-events-dead-letter-subscription"
  topic = google_pubsub_topic.events_dead_letter.name
  labels = {
    environment = local.environment
  }
}
