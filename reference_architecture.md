# Reference Architecture

## General Recomendations

* Set the ack deadline for your dead letter queues to a reasonably long amount of time. For the optimal user experience when using replay's dead letter review functionality, you should be able to poll a message, review it, and decide whether to move or discard it within the configured ack deadline.
