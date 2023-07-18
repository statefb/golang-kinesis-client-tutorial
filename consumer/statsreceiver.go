package main

import (
	"log"
	"os"
	"time"
)

type StatsReceiver struct {
	logger *log.Logger
}

func NewStatsReceiver() *StatsReceiver {
	logger := log.New(os.Stdout, "StatsReceiver: ", log.Lshortfile)
	return &StatsReceiver{
		logger: logger,
	}
}

func (r *StatsReceiver) Checkpoint() {
	r.logger.Println("Checkpoint passed")
}

func (r *StatsReceiver) EventToClient(inserted, retrieved time.Time) {
	r.logger.Println("inserted: ", inserted, " retrieved: ", retrieved)
}

func (r *StatsReceiver) EventsFromKinesis(num int, shardID string, lag time.Duration) {
	if lag > time.Duration(time.Duration.Seconds(0)) {
		r.logger.Println("num: ", num, " shardID: ", shardID, " lag: ", lag)
	}
}
