package main

import (
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	HttpServer http.Server
	CustomName string
}

func NewServer(addr string, opts ...Option) (*Server, error) {
	options := defaultServerOpts
	for _, opt := range opts {
		err := opt(&options)
		if err != nil {
			return nil, err
		}
	}

	srv := Server{
		HttpServer: http.Server{
			Addr:         fmt.Sprintf("%s:%d", addr, options.port),
			ReadTimeout:  options.readTimeout,
			WriteTimeout: options.readTimeout,
		},
		CustomName: options.serverName,
	}

	return &srv, nil
}

func main() {
	server, err := NewServer("127.0.0.1", WithPort(9000))
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(server.HttpServer.Addr) // 9000
	log.Println(server.CustomName)      // default

	srv2, err := NewServer("127.0.0.1", WithPort(1234), WithServerName("test_server"))
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(srv2.CustomName) // test_server
}
