package common

import (
	log "github.com/Sirupsen/logrus"
)

func noopRecoverLog() {
	if r := recover(); r != nil {
		log.Printf("Recovered: %v", r)
	}
}
