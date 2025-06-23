package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

type Database struct {
	db *sql.DB
}

func New(databasePath string) (*Database, error) {
	// Ensure data directory exists
	dir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure SQLite connection
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	database := &Database{db: db}

	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return database, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			balance TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trades (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			symbol TEXT NOT NULL,
			side TEXT NOT NULL,
			type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			price TEXT NOT NULL,
			fill_price TEXT NOT NULL DEFAULT '0',
			status TEXT NOT NULL,
			commission TEXT NOT NULL DEFAULT '0',
			alpaca_order_id TEXT,
			strategy TEXT NOT NULL,
			notes TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			filled_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		`CREATE TABLE IF NOT EXISTS portfolios (
			user_id INTEGER NOT NULL,
			symbol TEXT NOT NULL,
			quantity TEXT NOT NULL,
			average_price TEXT NOT NULL,
			current_value TEXT NOT NULL DEFAULT '0',
			unrealized_pl TEXT NOT NULL DEFAULT '0',
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (user_id, symbol),
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		`CREATE TABLE IF NOT EXISTS trading_signals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol TEXT NOT NULL,
			signal TEXT NOT NULL,
			strength REAL NOT NULL,
			price TEXT NOT NULL,
			strategy TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS market_data (
			symbol TEXT NOT NULL,
			price TEXT NOT NULL,
			volume INTEGER NOT NULL,
			high TEXT NOT NULL,
			low TEXT NOT NULL,
			open TEXT NOT NULL,
			close TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			PRIMARY KEY (symbol, timestamp)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_user_id ON trades (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades (symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_status ON trades (status)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_created_at ON trades (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_signals_symbol ON trading_signals (symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_signals_created_at ON trading_signals (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_market_data_symbol ON market_data (symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_market_data_timestamp ON market_data (timestamp)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	log.Println("Database migration completed successfully")
	return nil
}

// User operations
func (d *Database) CreateUser(user *models.User) error {
	query := `INSERT INTO users (username, email, balance, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?)`

	result, err := d.db.Exec(query, user.Username, user.Email, user.Balance.String(),
		user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}
	user.ID = id

	return nil
}

func (d *Database) GetUser(id int64) (*models.User, error) {
	query := `SELECT id, username, email, balance, created_at, updated_at 
			  FROM users WHERE id = ?`

	user := &models.User{}
	var balanceStr string

	err := d.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email,
		&balanceStr, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balance: %w", err)
	}
	user.Balance = balance

	return user, nil
}

func (d *Database) UpdateUser(user *models.User) error {
	query := `UPDATE users SET username = ?, email = ?, balance = ?, updated_at = ? 
			  WHERE id = ?`

	_, err := d.db.Exec(query, user.Username, user.Email, user.Balance.String(),
		user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Trade operations
func (d *Database) CreateTrade(trade *models.Trade) error {
	query := `INSERT INTO trades (user_id, symbol, side, type, quantity, price, 
			  status, commission, alpaca_order_id, strategy, notes, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := d.db.Exec(query, trade.UserID, trade.Symbol, trade.Side, trade.Type,
		trade.Quantity.String(), trade.Price.String(), trade.Status, trade.Commission.String(),
		trade.AlpacaOrderID, trade.Strategy, trade.Notes, trade.CreatedAt, trade.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create trade: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get trade ID: %w", err)
	}
	trade.ID = id

	return nil
}

func (d *Database) UpdateTrade(trade *models.Trade) error {
	query := `UPDATE trades SET fill_price = ?, status = ?, commission = ?, 
			  updated_at = ?, filled_at = ? WHERE id = ?`

	_, err := d.db.Exec(query, trade.FillPrice.String(), trade.Status,
		trade.Commission.String(), trade.UpdatedAt, trade.FilledAt, trade.ID)
	if err != nil {
		return fmt.Errorf("failed to update trade: %w", err)
	}

	return nil
}

func (d *Database) GetTradesByUser(userID int64, limit int) ([]*models.Trade, error) {
	query := `SELECT id, user_id, symbol, side, type, quantity, price, fill_price, 
			  status, commission, alpaca_order_id, strategy, notes, created_at, 
			  updated_at, filled_at FROM trades WHERE user_id = ? 
			  ORDER BY created_at DESC LIMIT ?`

	rows, err := d.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		trade := &models.Trade{}
		var quantityStr, priceStr, fillPriceStr, commissionStr string
		var filledAt sql.NullTime

		err := rows.Scan(&trade.ID, &trade.UserID, &trade.Symbol, &trade.Side, &trade.Type,
			&quantityStr, &priceStr, &fillPriceStr, &trade.Status, &commissionStr,
			&trade.AlpacaOrderID, &trade.Strategy, &trade.Notes, &trade.CreatedAt,
			&trade.UpdatedAt, &filledAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}

		// Parse decimal fields
		if trade.Quantity, err = decimal.NewFromString(quantityStr); err != nil {
			return nil, fmt.Errorf("failed to parse quantity: %w", err)
		}
		if trade.Price, err = decimal.NewFromString(priceStr); err != nil {
			return nil, fmt.Errorf("failed to parse price: %w", err)
		}
		if trade.FillPrice, err = decimal.NewFromString(fillPriceStr); err != nil {
			return nil, fmt.Errorf("failed to parse fill price: %w", err)
		}
		if trade.Commission, err = decimal.NewFromString(commissionStr); err != nil {
			return nil, fmt.Errorf("failed to parse commission: %w", err)
		}

		if filledAt.Valid {
			trade.FilledAt = &filledAt.Time
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// Portfolio operations
func (d *Database) UpsertPortfolio(portfolio *models.Portfolio) error {
	query := `INSERT OR REPLACE INTO portfolios (user_id, symbol, quantity, average_price, 
			  current_value, unrealized_pl, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query, portfolio.UserID, portfolio.Symbol,
		portfolio.Quantity.String(), portfolio.AveragePrice.String(),
		portfolio.CurrentValue.String(), portfolio.UnrealizedPL.String(),
		portfolio.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert portfolio: %w", err)
	}

	return nil
}

func (d *Database) GetPortfolioByUser(userID int64) ([]*models.Portfolio, error) {
	query := `SELECT user_id, symbol, quantity, average_price, current_value, 
			  unrealized_pl, updated_at FROM portfolios WHERE user_id = ? 
			  AND quantity != '0'`

	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio: %w", err)
	}
	defer rows.Close()

	var portfolios []*models.Portfolio
	for rows.Next() {
		portfolio := &models.Portfolio{}
		var quantityStr, avgPriceStr, currentValueStr, unrealizedPLStr string

		err := rows.Scan(&portfolio.UserID, &portfolio.Symbol, &quantityStr,
			&avgPriceStr, &currentValueStr, &unrealizedPLStr, &portfolio.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio: %w", err)
		}

		// Parse decimal fields
		if portfolio.Quantity, err = decimal.NewFromString(quantityStr); err != nil {
			return nil, fmt.Errorf("failed to parse quantity: %w", err)
		}
		if portfolio.AveragePrice, err = decimal.NewFromString(avgPriceStr); err != nil {
			return nil, fmt.Errorf("failed to parse average price: %w", err)
		}
		if portfolio.CurrentValue, err = decimal.NewFromString(currentValueStr); err != nil {
			return nil, fmt.Errorf("failed to parse current value: %w", err)
		}
		if portfolio.UnrealizedPL, err = decimal.NewFromString(unrealizedPLStr); err != nil {
			return nil, fmt.Errorf("failed to parse unrealized P&L: %w", err)
		}

		portfolios = append(portfolios, portfolio)
	}

	return portfolios, nil
}
