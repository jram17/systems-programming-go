package storage

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbpath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`        
	CREATE TABLE IF NOT EXISTS urls (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            code TEXT UNIQUE,
            url TEXT NOT NULL,
            clicks INTEGER DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
	`)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) GetNextID() (int64, error) {
	var nextID int64
	err := s.db.QueryRow("SELECT COALESCE(MAX(id), 0) + 1 FROM urls").Scan(&nextID)
	return nextID, err
}

func (s *Store) Save(code string, url string)(error){
	_,err:=s.db.Exec("INSERT INTO urls (code, url) VALUES (?, ?)", code, url)
	return err
}

func (s *Store) Get(code string) (string, error) {
	var url string
	err := s.db.QueryRow("SELECT url FROM urls where code = ?", code).Scan(&url)
	return url, err
}

func (s *Store) IncrementClicks(code string) error {
	_, err := s.db.Exec("UPDATE urls set clicks = clicks + 1 where code = ?", code)
	return err
}

func (s *Store) GetStats(code string) (url string, clicks int, createdAt time.Time, err error) {
	err = s.db.QueryRow(`
        SELECT url, clicks, created_at 
        FROM urls 
        WHERE code = ?
    `, code).Scan(&url, &clicks, &createdAt)
	return
}
