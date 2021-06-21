package crdb

import (
	"fmt"
	"time"
)

type crdbOptions struct {
	connMaxIdleTime *time.Duration
	connMaxLifetime *time.Duration
	minOpenConns    *int
	maxOpenConns    *int

	watchBufferLength    uint16
	revisionQuantization time.Duration
	gcWindow             time.Duration
}

const (
	errQuantizationTooLarge = "revision quantization (%s) must be less than GC window (%s)"

	defaultRevisionQuantization = 5 * time.Second
	defaultWatchBufferLength    = 128
)

type CRDBOption func(*crdbOptions)

func generateConfig(options []CRDBOption) (crdbOptions, error) {
	computed := crdbOptions{
		gcWindow:             24 * time.Hour,
		watchBufferLength:    defaultWatchBufferLength,
		revisionQuantization: defaultRevisionQuantization,
	}

	for _, option := range options {
		option(&computed)
	}

	// Run any checks on the config that need to be done
	if computed.revisionQuantization >= computed.gcWindow {
		return computed, fmt.Errorf(
			errQuantizationTooLarge,
			computed.revisionQuantization,
			computed.gcWindow,
		)
	}

	return computed, nil
}

// ConnMaxIdleTime is the duration after which an idle connection will be automatically closed by
// the health check.
// Default: no maximum
func ConnMaxIdleTime(idle time.Duration) CRDBOption {
	return func(po *crdbOptions) {
		po.connMaxIdleTime = &idle
	}
}

// ConnMaxLifetime is the duration since creation after which a connection will be automatically
// closed.
// Default: no maximum
func ConnMaxLifetime(lifetime time.Duration) CRDBOption {
	return func(po *crdbOptions) {
		po.connMaxLifetime = &lifetime
	}
}

// MinOpenConns is the minimum size of the connection pool. The health check will increase the
// number of connections to this amount if it had dropped below.
// Default: 0
func MinOpenConns(conns int) CRDBOption {
	return func(po *crdbOptions) {
		po.minOpenConns = &conns
	}
}

// MaxOpenConns is the maximum size of the connection pool.
// Default: no maximum
func MaxOpenConns(conns int) CRDBOption {
	return func(po *crdbOptions) {
		po.maxOpenConns = &conns
	}
}

// WatchBufferLength is the number of entries that can be stored in the watch buffer while awaiting
// read by the client.
// Default: 128
func WatchBufferLength(watchBufferLength uint16) CRDBOption {
	return func(po *crdbOptions) {
		po.watchBufferLength = watchBufferLength
	}
}

// RevisionQuantization is the time bucket size to which advertised revisions will be rounded.
// Default: 5s
func RevisionQuantization(bucketSize time.Duration) CRDBOption {
	return func(po *crdbOptions) {
		po.revisionQuantization = bucketSize
	}
}

// GCWindow is the maximum age of a passed revision that will be considered valid.
// Default: 24h
func GCWindow(window time.Duration) CRDBOption {
	return func(po *crdbOptions) {
		po.gcWindow = window
	}
}