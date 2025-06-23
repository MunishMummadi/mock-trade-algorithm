package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type TradeType string
type TradeStatus string
type OrderSide string

const (
	// Trade Types
	TradeTypeMarket TradeType = "market"
	TradeTypeLimit  TradeType = "limit"
	TradeTypeStop   TradeType = "stop"

	// Trade Status
	TradeStatusPending   TradeStatus = "pending"
	TradeStatusFilled    TradeStatus = "filled"
	TradeStatusCancelled TradeStatus = "cancelled"
	TradeStatusRejected  TradeStatus = "rejected"
	TradeStatusExpired   TradeStatus = "expired"

	// Order Sides
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

type Trade struct {
	ID            int64           `json:"id" db:"id"`
	UserID        int64           `json:"user_id" db:"user_id"`
	Symbol        string          `json:"symbol" db:"symbol"`
	Side          OrderSide       `json:"side" db:"side"`
	Type          TradeType       `json:"type" db:"type"`
	Quantity      decimal.Decimal `json:"quantity" db:"quantity"`
	Price         decimal.Decimal `json:"price" db:"price"`
	FillPrice     decimal.Decimal `json:"fill_price" db:"fill_price"`
	Status        TradeStatus     `json:"status" db:"status"`
	Commission    decimal.Decimal `json:"commission" db:"commission"`
	AlpacaOrderID string          `json:"alpaca_order_id" db:"alpaca_order_id"`
	Strategy      string          `json:"strategy" db:"strategy"`
	Notes         string          `json:"notes" db:"notes"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	FilledAt      *time.Time      `json:"filled_at" db:"filled_at"`
}

type TradingSignal struct {
	ID        int64           `json:"id" db:"id"`
	Symbol    string          `json:"symbol" db:"symbol"`
	Signal    string          `json:"signal"`   // BUY, SELL, HOLD
	Strength  float64         `json:"strength"` // 0-1 confidence
	Price     decimal.Decimal `json:"price" db:"price"`
	Strategy  string          `json:"strategy" db:"strategy"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

type MarketData struct {
	Symbol    string          `json:"symbol" db:"symbol"`
	Price     decimal.Decimal `json:"price" db:"price"`
	Volume    int64           `json:"volume" db:"volume"`
	High      decimal.Decimal `json:"high" db:"high"`
	Low       decimal.Decimal `json:"low" db:"low"`
	Open      decimal.Decimal `json:"open" db:"open"`
	Close     decimal.Decimal `json:"close" db:"close"`
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
}

func NewTrade(userID int64, symbol string, side OrderSide, tradeType TradeType, quantity, price decimal.Decimal, strategy string) *Trade {
	now := time.Now()
	return &Trade{
		UserID:    userID,
		Symbol:    symbol,
		Side:      side,
		Type:      tradeType,
		Quantity:  quantity,
		Price:     price,
		Status:    TradeStatusPending,
		Strategy:  strategy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (t *Trade) MarkFilled(fillPrice decimal.Decimal, commission decimal.Decimal) {
	now := time.Now()
	t.FillPrice = fillPrice
	t.Commission = commission
	t.Status = TradeStatusFilled
	t.FilledAt = &now
	t.UpdatedAt = now
}

func (t *Trade) Cancel() {
	t.Status = TradeStatusCancelled
	t.UpdatedAt = time.Now()
}

func (t *Trade) GetTotalCost() decimal.Decimal {
	if t.Status != TradeStatusFilled {
		return decimal.Zero
	}

	cost := t.Quantity.Mul(t.FillPrice)
	if t.Side == OrderSideBuy {
		return cost.Add(t.Commission)
	}
	return cost.Sub(t.Commission)
}

func (t *Trade) GetProfitLoss(currentPrice decimal.Decimal) decimal.Decimal {
	if t.Status != TradeStatusFilled {
		return decimal.Zero
	}

	if t.Side == OrderSideBuy {
		return currentPrice.Sub(t.FillPrice).Mul(t.Quantity).Sub(t.Commission)
	}
	return t.FillPrice.Sub(currentPrice).Mul(t.Quantity).Sub(t.Commission)
}
