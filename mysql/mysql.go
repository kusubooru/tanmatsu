package mysql

import (
	"context"
	"database/sql"
	"log"
	"time"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/kusubooru/tanmatsu"
)

type TanmatsuStore struct {
	*sql.DB
}

func NewTanmatsuStore(dataSource string) (*TanmatsuStore, error) {
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, err
	}
	if err := pingDatabase(db); err != nil {
		log.Fatalln("database ping attempts failed:", err)
	}
	return &TanmatsuStore{DB: db}, nil
}

func pingDatabase(db *sql.DB) (err error) {
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			return
		}
		log.Printf("database ping failed: %v, retry in 1s", err)
		time.Sleep(time.Second)
	}
	return
}

func (db *TanmatsuStore) GetImages(ctx context.Context, limit, offset int, orderAsc, isDone bool) ([]*tanmatsu.Image, error) {
	order := "DESC"
	if orderAsc {
		order = "ASC"
	}

	filter := ""
	if isDone {
		filter = " AND img.done = 1 "
	}

	var query = `
	SELECT
	  img.id,
	  img.url,
	  img.done,
	  img.kuid,
	  img.created,
	  m.player,
	  m.channel
	FROM images img
	  JOIN messages m ON img.message_id=m.id` + filter + `
	  ORDER BY created ` + order + ` LIMIT ?, ?`

	rows, err := db.Query(query, offset, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
			return
		}
	}()

	images := []*tanmatsu.Image{}
	for rows.Next() {
		img := tanmatsu.Image{}
		err := rows.Scan(
			&img.ID,
			&img.URL,
			&img.Done,
			&img.Kuid,
			&img.Created,
			&img.Player,
			&img.Channel,
		)
		if err != nil {
			return nil, err
		}
		images = append(images, &img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return images, nil
}

func (db *TanmatsuStore) createImage(ctx context.Context, img *tanmatsu.Image) error {
	var messageQuery = `
	INSERT INTO messages(
		message,
		player,
		channel
	) values(
		?,
		?,
		?
	)`
	res, err := db.ExecContext(ctx, messageQuery, "", img.Player, img.Channel)
	if err != nil {
		return err
	}
	messageID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	var imageQuery = `
	INSERT INTO images(
		url,
		done,
		kuid,
		message_id
	) values(
		?,
		?,
		?,
		?
	)`

	res, err = db.ExecContext(ctx, imageQuery, img.URL, img.Done, img.Kuid, messageID)
	if err != nil {
		return err
	}
	imageID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	img.ID = imageID

	return nil
}
