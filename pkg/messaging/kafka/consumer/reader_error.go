package consumer

import (
	"errors"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// kafkaErrorType represents the category of Kafka error
type kafkaErrorType int

const (
	errorTypeTimeout kafkaErrorType = iota
	errorTypeFatal
	errorTypeTopicNotFound
	errorTypeBrokerConnection
	errorTypeLeaderElection
	errorTypeRetriable
	errorTypeNonKafka
)

// readerError wraps the original error with additional classification information
type readerError struct {
	err         error // original error
	errorType   kafkaErrorType
	errorKey    string // unique key for error tracking
	description string // human-readable description
}

// Error implements the error interface
func (e *readerError) Error() string {
	if e.description != "" {
		return fmt.Sprintf("%s: %v", e.description, e.err)
	}
	return e.err.Error()
}

// Unwrap returns the underlying error for errors.Is and errors.As
func (e *readerError) Unwrap() error {
	return e.err
}

// wrapReaderError analyzes a Kafka error and wraps it in readerError with classification
// Returns nil if there is no error
func wrapReaderError(err error) *readerError {
	if err == nil {
		return nil
	}

	var kafkaErr kafka.Error
	if !errors.As(err, &kafkaErr) {
		return &readerError{
			err:         err,
			errorType:   errorTypeNonKafka,
			errorKey:    "non_kafka_error",
			description: "non-Kafka error occurred",
		}
	}

	// Check for timeout
	if kafkaErr.IsTimeout() {
		return &readerError{
			err:         err,
			errorType:   errorTypeTimeout,
			errorKey:    "",
			description: "",
		}
	}

	// Check for fatal errors
	if kafkaErr.IsFatal() {
		return &readerError{
			err:         err,
			errorType:   errorTypeFatal,
			errorKey:    "",
			description: "fatal kafka error - consumer instance is no longer operable",
		}
	}

	// Classify by error code
	switch kafkaErr.Code() {
	case kafka.ErrUnknownTopicOrPart:
		return &readerError{
			err:         err,
			errorType:   errorTypeTopicNotFound,
			errorKey:    "topic_not_found",
			description: "topic not available, waiting for topic creation",
		}

	case kafka.ErrTransport, kafka.ErrAllBrokersDown, kafka.ErrNetworkException:
		return &readerError{
			err:         err,
			errorType:   errorTypeBrokerConnection,
			errorKey:    "broker_connection",
			description: "broker connection issue, retrying",
		}

	case kafka.ErrLeaderNotAvailable, kafka.ErrNotLeaderForPartition:
		return &readerError{
			err:         err,
			errorType:   errorTypeLeaderElection,
			errorKey:    "leader_election",
			description: "partition leader changing, retrying",
		}
	}

	// Check for other retriable errors
	if kafkaErr.IsRetriable() {
		return &readerError{
			err:         err,
			errorType:   errorTypeRetriable,
			errorKey:    "retriable_error",
			description: "retriable kafka error, retrying",
		}
	}

	// Unknown error type
	return &readerError{
		err:         err,
		errorType:   errorTypeNonKafka,
		errorKey:    "unknown_error",
		description: "unknown kafka error",
	}
}

// isFatal returns true if the error is fatal and consumer should stop
func (e *readerError) isFatal() bool {
	return e.errorType == errorTypeFatal
}

// isTimeout returns true if the error is a timeout
func (e *readerError) isTimeout() bool {
	return e.errorType == errorTypeTimeout
}

// isTemporary returns true if the error is temporary and we should retry
func (e *readerError) isTemporary() bool {
	switch e.errorType {
	case errorTypeTopicNotFound,
		errorTypeBrokerConnection,
		errorTypeLeaderElection,
		errorTypeRetriable:
		return true
	default:
		return false
	}
}
