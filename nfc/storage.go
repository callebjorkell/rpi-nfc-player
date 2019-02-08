package nfc

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/buntdb"
)

type DB struct {
	instance *buntdb.DB
}

func (db *DB) Close() error {
	return db.instance.Close()
}

func (db *DB) StoreCard(c Card) error {
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

func (db *DB) ReadCard(id int) (Card, error) {
	var c Card
	err := db.instance.View(func(tx *buntdb.Tx) error {
		s, err := tx.Get(getCardKey(id))
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(s), &c)
	})
	return c, err
}

func (db *DB) DeleteCard(id int) error {
	return db.instance.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(getCardKey(id))
		return err
	})
}



func getCardKey(id int) string {
	return fmt.Sprintf("card:%v", id)
}

func NewDB() (*DB, error) {
	db, err := buntdb.Open("tracks.db")
	if err != nil {
		return nil, err
	}
	return &DB{instance: db}, nil
}
