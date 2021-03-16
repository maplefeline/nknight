package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var sigint chan os.Signal

func waitShutdown(e *echo.Echo, idleConnsClosed chan<- interface{}) {
	defer close(idleConnsClosed)

	sigint = make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	defer signal.Stop(sigint)

	<-sigint
	log.Info("received shutdown signal")

	idleError("HTTP server shutdown:", e.Shutdown(context.Background()))
}

func listenAndServe(addr string, idleConnsClosed chan<- interface{}) {
	e := apiHandler()
	go waitShutdown(e, idleConnsClosed)

	e.Use(middleware.Logger())

	idleError("HTTP server end:", e.Start(addr))
}

// Open open.
func Open(addr string) {
	idleConnsClosed := make(chan interface{})
	go listenAndServe(addr, idleConnsClosed)
	<-idleConnsClosed
}

func idle() {
	idleError("agent idle complete:", agentIdle())
	idleError("game idle complete:", gameIdle())
}

// Close close.
func Close() error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func main() {
	defer func() {
		idleError("close server:", Close())
	}()
	flag.Parse()
	go func() {
		for {
			idle()
		}
	}()
	Open(":8080")
}
