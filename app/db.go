package app

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
)

var db *sql.DB

func (s *Settings) GetDatabase() *sql.DB {
	if db != nil {
		return db
	}

	log.Print("mysql DSN ", s.Db.Dsn, "?", s.Db.Opt)
	if handle, err := sql.Open("sqlite3", s.Db.Dsn+"?"+s.Db.Opt); err != nil {
		s.Fatal(err)
	} else {
		db = handle
		s.initDb(db)
	}
	return db
}

func (s *Settings) CloseDatabase() {
	if db != nil {
		db.Close()
	}
}

func (s *Settings) initDb(db *sql.DB) {
	sqlStmt := "CREATE TABLE IF NOT EXISTS `feed` (" +
		"`ts` TEXT NOT NULL," +
		"`sent` TEXT NULL," +
		"`url` TEXT PRIMARY KEY," +
		"`title` TEXT NOT NULL," +
		"`category` TEXT NOT NULL," +
		"`teaser` TEXT NOT NULL," +
		"`content` TEXT NOT NULL," +
		"`enclosure` TEXT NOT NULL" +
		")"
	if _, err := db.Exec(sqlStmt); err != nil {
		s.Fatal(err)
	}

	owa := time.Now().AddDate(0, 0, -21)
	sdel := "DELETE FROM `feed` WHERE `sent` IS NOT NULL AND `sent`<?"
	if _, err := db.Exec(sdel, owa.Format(time.RFC3339)); err != nil {
		s.Fatal(err)
	}
}

func (s *Settings) StoreItem(item *gofeed.Item) bool {
	db := s.GetDatabase()

	categories := strings.Join(item.Categories, ";")

	enclosure, err := json.Marshal(item.Image)
	if err != nil {
		s.Log(err)
	}
	sql := "INSERT OR IGNORE INTO `feed` (`ts`,`url`,`title`,`teaser`,`content`,`category`,`enclosure`) VALUES (?,?,?,?,?,?,?)"
	if rslt, err := db.Exec(
		sql,
		item.PublishedParsed.Format(time.RFC3339),
		item.Link,
		item.Title,
		item.Description,
		item.Content,
		categories,
		enclosure,
	); err != nil {
		s.Log(err)
	} else if ra, err := rslt.RowsAffected(); err != nil {
		s.Log(err)
	} else {
		return ra > 0
	}
	return false
}

func (s *Settings) GetUnsent() *gofeed.Item {
	db := s.GetDatabase()
	sql := "SELECT `ts`,`url`,`title`,`category`,`teaser`,`content`,`enclosure` FROM `feed` WHERE `sent` IS NULL ORDER BY `ts` ASC LIMIT 1"
	row := db.QueryRow(sql)
	item := gofeed.Item{}
	categories := ""
	enclosure := ""
	if err := row.Scan(
		&item.Published,
		&item.Link,
		&item.Title,
		&categories,
		&item.Description,
		&item.Content,
		&enclosure,
	); err != nil {
		s.Log(err)
		return nil
	}
	item.Categories = strings.Split(categories, ";")

	if err := json.Unmarshal([]byte(enclosure), &item.Image); err != nil {
		s.Log(err)
	}
	return &item
}

func (s *Settings) MarkSent(item *gofeed.Item) {
	db := s.GetDatabase()
	sql := "UPDATE `feed` SET `sent`=datetime() WHERE `url`=?"
	if _, err := db.Exec(sql, item.Link); err != nil {
		s.Log(err)
	}
}

func (s *Settings) MarkError(item *gofeed.Item, err error) {
	db := s.GetDatabase()
	sql := "UPDATE `feed` SET `sent`=? WHERE `url`=?"
	if _, err := db.Exec(sql, err.Error(), item.Link); err != nil {
		s.Log(err)
	}
}
