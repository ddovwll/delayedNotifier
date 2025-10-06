package data

import (
	"fmt"
	"os"

	"github.com/wb-go/wbf/dbpg"
)

func InitDb() (*dbpg.DB, error) {
	sqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(sqlInfo, []string{}, opts)
	if err != nil {
		return nil, err
	}

	return db, nil
}
