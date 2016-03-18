package common

import (
	log "github.com/Sirupsen/logrus"
)

func NoopRecoverLog() {
	if r := recover(); r != nil {
		log.Printf("Recovered: %v", r)
	}
}

func RecoverAndDo(fn func()) {
	if r := recover(); r != nil {
		log.Printf("Recovered: %v", r)
		fn()
	}

}
