package main

import (
	log "github.com/sirupsen/logrus"
)

func removeCard(cardId string) {
	if cardId == "" {
		id, err := readSingleCard()
		if err != nil {
			log.Fatal(err)
		}
		cardId = id
	}

	if err := db.DeleteCard(cardId); err != nil {
		log.Warnf("Could not remove card %v: %v", cardId, err.Error())
	}
}
