package mysql

func (db *TanmatsuStore) createSchema() error {
	if _, err := db.Exec(tableMessages); err != nil {
		return err
	}
	if _, err := db.Exec(tableImages); err != nil {
		return err
	}
	return nil
}

func (db *TanmatsuStore) dropSchema() error {
	if _, err := db.Exec(`DROP TABLE images`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP TABLE messages`); err != nil {
		return err
	}
	return nil
}

const (
	tableMessages = `
CREATE TABLE IF NOT EXISTS messages (
	id SERIAL,
	message TEXT NOT NULL,
	player VARCHAR(255) NOT NULL,
	channel VARCHAR(255) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
)`

	tableImages = `
CREATE TABLE IF NOT EXISTS images (
	id SERIAL,
	url VARCHAR(2000) NOT NULL,
	done BOOL NOT NULL DEFAULT 0,
	kuid INT NOT NULL DEFAULT 0,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	message_id BIGINT UNSIGNED NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (message_id) REFERENCES messages(id)
)`
)
