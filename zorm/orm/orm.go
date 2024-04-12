package orm

import (
	"database/sql"
	"time"
)

type ZDb struct {
	db *sql.DB
}

type DbSession struct {
	db        *ZDb
	tableName string
}

func Open(driverName, source string) *ZDb {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetConnMaxIdleTime(time.Minute * 1)

	zdb := &ZDb{db: db}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return zdb
}

func (db *ZDb) New() *DbSession {
	return &DbSession{
		db: db,
	}
}

func (db *ZDb) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

func (session *DbSession) Table(name string) *DbSession {
	session.tableName = name
	return session
}

func (session *DbSession) Insert(data any) {

}
