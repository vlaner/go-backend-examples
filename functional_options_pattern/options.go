package main

import (
	"errors"
	"time"
)

type serverOpts struct {
	port         int
	readTimeout  time.Duration
	writeTimeout time.Duration
	serverName   string
}

type Option func(options *serverOpts) error

// or func NewDefaultServerOptions ...
var defaultServerOpts = serverOpts{
	port:         8080,
	readTimeout:  3 * time.Second,
	writeTimeout: 10 * time.Second,
	serverName:   "default",
}

func WithPort(port int) Option {
	return func(options *serverOpts) error {
		if port < 0 {
			return errors.New("port must be positive int")
		}

		options.port = port

		return nil
	}
}

func WithReadTimeout(readTimeout time.Duration) Option {
	return func(options *serverOpts) error {
		if readTimeout <= 0 {
			return errors.New("read timeout must be positive and greater than 0")
		}

		options.readTimeout = readTimeout

		return nil
	}
}

func WithWriteTimeout(writeTimeout time.Duration) Option {
	return func(options *serverOpts) error {
		if writeTimeout <= 0 {
			return errors.New("write timeout must be positive and greater than 0")
		}

		options.writeTimeout = writeTimeout

		return nil
	}
}

func WithServerName(serverName string) Option {
	return func(options *serverOpts) error {
		options.serverName = serverName
		return nil
	}
}
