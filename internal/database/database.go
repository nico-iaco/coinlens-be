package database

import (
	"coinlens-be/internal/models"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(connString string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Connected to database")
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) DeleteCoin(ctx context.Context, id string) error {
	result, err := db.Pool.Exec(ctx, "DELETE FROM coins WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete coin: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("coin not found")
	}
	return nil
}

func (db *DB) CreateCoin(ctx context.Context, coin *models.Coin) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO coins (id, name, description, year, country)
		VALUES ($1, $2, $3, $4, $5)
	`, coin.ID, coin.Name, coin.Description, coin.Year, coin.Country)
	return err
}

func (db *DB) UpdateCoinMetadata(ctx context.Context, id string, analysis *models.CoinAnalysis) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE coins 
		SET name = $1, description = $2, year = $3, country = $4
		WHERE id = $5
	`, analysis.Name, analysis.Description, analysis.Year, analysis.Country, id)
	if err != nil {
		return fmt.Errorf("failed to update coin metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("coin not found")
	}
	return nil
}
