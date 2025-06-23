package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type User struct {
	ID        int64           `json:"id" db:"id"`
	Username  string          `json:"username" db:"username"`
	Email     string          `json:"email" db:"email"`
	Balance   decimal.Decimal `json:"balance" db:"balance"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

type Portfolio struct {
	UserID       int64           `json:"user_id" db:"user_id"`
	Symbol       string          `json:"symbol" db:"symbol"`
	Quantity     decimal.Decimal `json:"quantity" db:"quantity"`
	AveragePrice decimal.Decimal `json:"average_price" db:"average_price"`
	CurrentValue decimal.Decimal `json:"current_value" db:"current_value"`
	UnrealizedPL decimal.Decimal `json:"unrealized_pl" db:"unrealized_pl"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

type UserStats struct {
	UserID         int64           `json:"user_id"`
	TotalTrades    int64           `json:"total_trades"`
	WinningTrades  int64           `json:"winning_trades"`
	LosingTrades   int64           `json:"losing_trades"`
	WinRate        float64         `json:"win_rate"`
	TotalPL        decimal.Decimal `json:"total_pl"`
	MaxDrawdown    decimal.Decimal `json:"max_drawdown"`
	SharpeRatio    float64         `json:"sharpe_ratio"`
	PortfolioValue decimal.Decimal `json:"portfolio_value"`
}

func NewUser(username, email string, initialBalance decimal.Decimal) *User {
	now := time.Now()
	return &User{
		Username:  username,
		Email:     email,
		Balance:   initialBalance,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (u *User) UpdateBalance(amount decimal.Decimal) {
	u.Balance = u.Balance.Add(amount)
	u.UpdatedAt = time.Now()
}

func (u *User) CanAfford(amount decimal.Decimal) bool {
	return u.Balance.GreaterThanOrEqual(amount)
}

func (p *Portfolio) UpdatePosition(quantity, price decimal.Decimal) {
	if p.Quantity.IsZero() {
		// New position
		p.Quantity = quantity
		p.AveragePrice = price
	} else if quantity.Sign() == p.Quantity.Sign() {
		// Adding to existing position
		totalCost := p.Quantity.Mul(p.AveragePrice).Add(quantity.Mul(price))
		p.Quantity = p.Quantity.Add(quantity)
		if !p.Quantity.IsZero() {
			p.AveragePrice = totalCost.Div(p.Quantity)
		}
	} else {
		// Reducing or closing position
		p.Quantity = p.Quantity.Add(quantity)
		if p.Quantity.IsZero() {
			p.AveragePrice = decimal.Zero
		}
	}
	p.UpdatedAt = time.Now()
}

func (p *Portfolio) CalculateUnrealizedPL(currentPrice decimal.Decimal) {
	if p.Quantity.IsZero() {
		p.UnrealizedPL = decimal.Zero
		return
	}

	currentValue := p.Quantity.Mul(currentPrice)
	costBasis := p.Quantity.Mul(p.AveragePrice)
	p.UnrealizedPL = currentValue.Sub(costBasis)
	p.CurrentValue = currentValue
}
