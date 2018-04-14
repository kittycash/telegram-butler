package auction_butler

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/jmoiron/sqlx"
)

type DB struct {
	*sqlx.DB
}

func NewDB(config *DatabaseConfig) (*DB, error) {
	if config == nil {
		errors.New("config should not be nil in NewDB()")
	}
	db, err := sqlx.Open(config.Driver, config.Source)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) GetUser(id int) *User {
	var user User
	err := db.Get(&user, db.Rebind("select * from botuser where id=?"), id)
	if err == sql.ErrNoRows {
		return nil
	}

	if err != nil {
		panic(err)
		return nil
	}

	user.exists = true
	return &user
}

func (db *DB) GetUserByName(name string) *User {
	var user User
	err := db.Get(&user, db.Rebind("select * from botuser where username=?"), name)

	if err == sql.ErrNoRows {
		return nil
	}

	if err != nil {
		panic(err)
		return nil
	}

	user.exists = true
	return &user
}

func (db *DB) GetUserByNameOrId(identifier string) *User {
	var user User
	var err error

	// Check if string is an integer
	// If its an int then check for user id
	// otherwise check for username
	if _, err := strconv.Atoi(identifier); err == nil {
		err = db.Get(&user, db.Rebind("select * from botuser where id=?"), identifier)
	} else {
		err = db.Get(&user, db.Rebind("select * from botuser where username=?"), identifier)
	}

	// @TODO improve this check
	if err == sql.ErrNoRows || user.ID == 0 {
		return nil
	}

	if err != nil {
		panic(err)
		return nil
	}

	user.exists = true
	return &user
}

func (db *DB) GetUsers(banned bool) ([]User, error) {
	var users []User

	err := db.Select(&users, db.Rebind("select * from botuser where banned = ? order by username"), banned)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (db *DB) GetAdmins() ([]User, error) {
	var users []User

	err := db.Select(&users, db.Rebind("select * from botuser where admin=true"))

	if err != nil {
		return nil, err
	}

	return users, nil
}

func (db *DB) GetUserCount(banned bool) (int, error) {
	var count int

	err := db.Get(&count, db.Rebind("select count(*) from botuser where banned = ?"), banned)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (db *DB) GetCurrentAuction() *Auction {
	var auction Auction

	err := db.Get(&auction, db.Rebind("select * from auction where ended=false and end_time>now() order by id asc limit 1"))
	if err == sql.ErrNoRows {
		return nil
	}

	if err != nil {
		panic(err)
		return nil
	}

	auction.exists = true

	return &auction
}

func (db *DB) PutAuction(auction *Auction) error {
	var err error
	if auction.exists {
		_, err = db.Exec(db.Rebind(`
			update auction set end_time=?, bid_val=?, bid_type=?, ended=?, bid_msg_id=? where id=?`),
			auction.EndTime.Time, auction.BidVal, auction.BidType, auction.Ended, auction.BidMessageID, auction.ID)
	} else {
		_, err = db.Exec(db.Rebind(`
		insert into auction (
			end_time, bid_val, bid_type
		) values (?, ?, ?)`),
			auction.EndTime.Time, 0, "",
		)
	}

	return err
}

func (db *DB) PutUser(u *User) error {
	if u.exists {
		_, err := db.Exec(db.Rebind(`
			update botuser
				set username = ?,
				first_name = ?,
				last_name = ?,
				banned = ?,
				admin = ?
			where id = ?`),
			u.UserName,
			u.FirstName,
			u.LastName,
			u.Banned,
			u.Admin,
			u.ID,
		)
		return err
	} else {
		_, err := db.Exec(db.Rebind(`
			insert into botuser (
				id, username, first_name, last_name,
				banned, admin
			) values (?, ?, ?, ?, ?, ?)`),
			u.ID,
			u.UserName,
			u.FirstName,
			u.LastName,
			u.Banned,
			u.Admin,
		)
		if err == nil {
			u.exists = true
		}
		return err
	}
}
