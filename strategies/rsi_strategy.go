package strategies

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/alpaca"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

// RSIStrategy implements a Relative Strength Index strategy
type RSIStrategy struct {
	BaseStrategy
	period          int
	oversoldLevel   float64
	overboughtLevel float64
}

// NewRSIStrategy creates a new RSI strategy
func NewRSIStrategy(period int, oversoldLevel, overboughtLevel float64) *RSIStrategy {
	return &RSIStrategy{
		BaseStrategy: BaseStrategy{
			name:        "RSI Strategy",
			description: "Relative Strength Index momentum strategy",
		},
		period:          period,
		oversoldLevel:   oversoldLevel,
		overboughtLevel: overboughtLevel,
	}
}

// Analyze implements the Strategy interface
func (r *RSIStrategy) Analyze(symbol string, bars []alpaca.MockBar, currentPrice decimal.Decimal) *models.TradingSignal {
	if len(bars) < r.period+1 {
		return nil
	}

	prices := ExtractPrices(bars)

	// Calculate RSI
	rsi := CalculateRSI(prices, r.period)
	if len(rsi) < 2 {
		return nil
	}

	// Get current and previous RSI values
	currentRSI := rsi[len(rsi)-1].InexactFloat64()
	prevRSI := rsi[len(rsi)-2].InexactFloat64()

	// Determine signal
	var signal string
	var strength float64

	if prevRSI > r.oversoldLevel && currentRSI <= r.oversoldLevel {
		// RSI crossed below oversold level - potential buy signal
		signal = "BUY"
		// Strength increases as RSI gets lower (more oversold)
		strength = (r.oversoldLevel - currentRSI) / r.oversoldLevel
		if strength > 1.0 {
			strength = 1.0
		}
		// Minimum strength for oversold conditions
		if strength < 0.6 {
			strength = 0.6
		}
	} else if prevRSI < r.overboughtLevel && currentRSI >= r.overboughtLevel {
		// RSI crossed above overbought level - potential sell signal
		signal = "SELL"
		// Strength increases as RSI gets higher (more overbought)
		strength = (currentRSI - r.overboughtLevel) / (100 - r.overboughtLevel)
		if strength > 1.0 {
			strength = 1.0
		}
		// Minimum strength for overbought conditions
		if strength < 0.6 {
			strength = 0.6
		}
	} else if currentRSI < 20 {
		// Extremely oversold condition
		signal = "BUY"
		strength = 0.9
	} else if currentRSI > 80 {
		// Extremely overbought condition
		signal = "SELL"
		strength = 0.9
	} else {
		// No clear signal
		return nil
	}

	return &models.TradingSignal{
		Symbol:    symbol,
		Signal:    signal,
		Strength:  strength,
		Price:     currentPrice,
		Strategy:  r.GetName(),
		CreatedAt: time.Now(),
	}
}
