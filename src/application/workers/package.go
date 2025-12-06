// Package workers provides background workers for async job processing.
// Workers consume messages from NATS queue and execute corresponding use cases.
//
// Components:
// - DraftGenerationWorker: Processes draft generation requests from queue
//
// Workers are designed to run as goroutines within the monolith application,
// providing async processing without requiring separate service deployments.
package workers
