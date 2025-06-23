package strategies

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/alpaca"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

// SMAStrategy implements a Simple Moving Average crossover strategy
type SMAStrategy struct {
	BaseStrategy
	shortPeriod int
	longPeriod  int
}

// NewSMAStrategy creates a new SMA strategy
func NewSMAStrategy(shortPeriod, longPeriod int) *SMAStrategy {
	return &SMAStrategy{
		BaseStrategy: BaseStrategy{
			name:        "SMA Crossover",
			description: "Simple Moving Average crossover strategy",
		},
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
	}
}

// Analyze implements the Strategy interface
func (s *SMAStrategy) Analyze(symbol string, bars []alpaca.MockBar, currentPrice decimal.Decimal) *models.TradingSignal {
	if len(bars) < s.longPeriod {
		return nil
	}

	prices := ExtractPrices(bars)

	// Calculate short and long SMAs
	shortSMA := CalculateSMA(prices, s.shortPeriod)
	longSMA := CalculateSMA(prices, s.longPeriod)

	if len(shortSMA) < 2 || len(longSMA) < 2 {
		return nil
	}

	// Get the latest values
	currentShortSMA := shortSMA[len(shortSMA)-1]
	currentLongSMA := longSMA[len(longSMA)-1]
	prevShortSMA := shortSMA[len(shortSMA)-2]
	prevLongSMA := longSMA[len(longSMA)-2]

	// Determine signal
	var signal string
	var strength float64

	// Check for crossover
	if prevShortSMA.LessThanOrEqual(prevLongSMA) && currentShortSMA.GreaterThan(currentLongSMA) {
		// Bullish crossover - short SMA crosses above long SMA
		signal = "BUY"
		// Calculate strength based on the magnitude of the crossover
		diff := currentShortSMA.Sub(currentLongSMA).Div(currentLongSMA).Abs()
		strength = 0.7 + (diff.InexactFloat64() * 100 * 3) // Scale and limit strength
		if strength > 1.0 {
			strength = 1.0
		}
	} else if prevShortSMA.GreaterThanOrEqual(prevLongSMA) && currentShortSMA.LessThan(currentLongSMA) {
		// Bearish crossover - short SMA crosses below long SMA
		signal = "SELL"
		diff := currentLongSMA.Sub(currentShortSMA).Div(currentLongSMA).Abs()
		strength = 0.7 + (diff.InexactFloat64() * 100 * 3)
		if strength > 1.0 {
			strength = 1.0
		}
	} else {
		// No clear signal
		return nil
	}

	return &models.TradingSignal{
		Symbol:    symbol,
		Signal:    signal,
		Strength:  strength,
		Price:     currentPrice,
		Strategy:  s.GetName(),
		CreatedAt: time.Now(),
	}
}
