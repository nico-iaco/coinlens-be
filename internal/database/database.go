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
		INSERT INTO coins (id, name, description, year, country, universal_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, coin.ID, coin.Name, coin.Description, coin.Year, coin.Country, coin.UniversalID)
	return err
}

func (db *DB) UpdateCoinMetadata(ctx context.Context, id string, analysis *models.CoinAnalysis) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE coins 
		SET name = $1, description = $2, year = $3, country = $4, universal_id = $5
		WHERE id = $6
	`, analysis.Name, analysis.Description, analysis.Year, analysis.Country, analysis.UniversalID, id)
	if err != nil {
		return fmt.Errorf("failed to update coin metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("coin not found")
	}
	return nil
}

func (db *DB) GetCoinsByUniversalID(ctx context.Context, universalID string) ([]models.Coin, error) {
	rows, err := db.Pool.Query(ctx, "SELECT id, name, description, year, country, universal_id, created_at FROM coins WHERE universal_id = $1 ORDER BY created_at DESC", universalID)
	if err != nil {
		return nil, fmt.Errorf("failed to search coins by universal id: %w", err)
	}
	defer rows.Close()

	var coins []models.Coin
	for rows.Next() {
		var c models.Coin
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Year, &c.Country, &c.UniversalID, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan coin: %w", err)
		}
		coins = append(coins, c)
	}
	return coins, nil
}
