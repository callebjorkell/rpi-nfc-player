package main

import (
	"encoding/json"
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	"github.com/tidwall/buntdb"
)

type DB struct {
	instance *buntdb.DB
}

// Get a DB, panicking on any error
func GetDB() *DB {
	db, err := buntdb.Open("tracks.db")
	if err != nil {
		panic(err)
	}
	conf := buntdb.Config{}
	err = db.ReadConfig(&conf)
	if err != nil {
		panic(err)
	}
	conf.SyncPolicy = buntdb.Always
	err = db.SetConfig(conf)
	if err != nil {
		panic(err)
	}
	return &DB{instance: db}
}

func (db *DB) Close() error {
	return db.instance.Close()
}

func (db *DB) StoreCard(c *sonos.CardInfo) error {
	return db.instance.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(c)
		if err != nil {
			return err
		}
		if _, _, err := tx.Set(getCardKey(c.ID), string(data), nil); err != nil {
			return err
		}
		return nil
	})
}

func (db *DB) ReadAll() (*[]sonos.CardInfo, error) {
	var cards []sonos.CardInfo
	err := db.instance.View(func(tx *buntdb.Tx) error {
		var shitHappened error
		err := tx.Ascend("", func(key, value string) bool {
			var c sonos.CardInfo
			shitHappened = json.Unmarshal([]byte(value), &c)
			if shitHappened != nil {
				return false
			}

			cards = append(cards, c)
			return true
		})
		if err != nil {
			return err
		}
		return shitHappened
	})
	return &cards, err
}

func (db *DB) ReadCard(id string) (sonos.CardInfo, error) {
	var c sonos.CardInfo
	err := db.instance.View(func(tx *buntdb.Tx) error {
		s, err := tx.Get(getCardKey(id))
		if err != nil {
			if err == buntdb.ErrNotFound {
				return fmt.Errorf("card %v has not been provisioned", id)
			}
			return err
		}
		return json.Unmarshal([]byte(s), &c)
	})
	return c, err
}

func (db *DB) DeleteCard(id string) error {
	return db.instance.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(getCardKey(id))
		return err
	})
}

func getCardKey(id string) string {
	return fmt.Sprintf("card:%v", id)
}
