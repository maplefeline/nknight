package main

import (
	"errors"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/apex/log"
	uuid "github.com/satori/go.uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

var placeHolder uuid.UUID

func init() {
	placeholder, err := uuid.FromString("f9a87c7e3f4f11eb99b58c8590001d9d")
	if err != nil {
		log.WithError(err).Fatal("failed to parse placeholder uuid")
	}
	placeHolder = placeholder

	dbname, ok := os.LookupEnv("PGDATABASE")
	if !ok {
		dbname = "test"
	}
	connStr := strings.Join([]string{"dbname", dbname}, "=")

	database, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Silent),
		QueryFields: true,
	})
	if err != nil {
		log.WithError(err).WithField("connStr", connStr).Fatal("failed to connect database")
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.WithError(err).Fatal("error")
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	database.AutoMigrate(&Board{}, &Game{})
	if err := database.Error; err != nil {
		log.WithError(err).Fatal("error")
	}

	db = database

	// board, err := makeBoard(&initialBoard)
	// if err != nil {
	// 	log.WithError(err).Fatal("error")
	// }
	// if err := board.lookahead(true); err != nil {
	// 	log.WithError(err).Fatal("error")
	// }
}

func idleError(message string, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	e := err
	for errors.Unwrap(e) != nil {
		e = errors.Unwrap(e)
	}
	if e.Error() == "sql: database is closed" {
		time.Sleep(1 * time.Second)
		return
	}
	log.WithField("type", reflect.TypeOf(err)).WithError(err).Error(message)
	panic(err)
}
