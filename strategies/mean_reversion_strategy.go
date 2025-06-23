package strategies

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/alpaca"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

// MeanReversionStrategy implements a mean reversion strategy using Bollinger Bands
type MeanReversionStrategy struct {
	BaseStrategy
	period             int
	standardDeviations float64
}

// NewMeanReversionStrategy creates a new mean reversion strategy
func NewMeanReversionStrategy(period int, standardDeviations float64) *MeanReversionStrategy {
	return &MeanReversionStrategy{
		BaseStrategy: BaseStrategy{
			name:        "Mean Reversion",
			description: "Bollinger Bands mean reversion strategy",
		},
		period:             period,
		standardDeviations: standardDeviations,
	}
}

// Analyze implements the Strategy interface
func (m *MeanReversionStrategy) Analyze(symbol string, bars []alpaca.MockBar, currentPrice decimal.Decimal) *models.TradingSignal {
	if len(bars) < m.period {
		return nil
	}

	prices := ExtractPrices(bars)

	// Calculate Bollinger Bands
	upper, middle, lower := CalculateBollingerBands(prices, m.period, m.standardDeviations)
	if len(upper) == 0 || len(middle) == 0 || len(lower) == 0 {
		return nil
	}

	// Get the latest values
	currentUpper := upper[len(upper)-1]
	currentMiddle := middle[len(middle)-1]
	currentLower := lower[len(lower)-1]

	// Calculate where current price is within the bands
	bandWidth := currentUpper.Sub(currentLower)
	if bandWidth.IsZero() {
		return nil
	}

	// Determine signal based on position relative to bands
	var signal string
	var strength float64

	if currentPrice.LessThan(currentLower) {
		// Price is below lower band - potential buy signal (oversold)
		signal = "BUY"
		// Calculate how far below the lower band
		distance := currentLower.Sub(currentPrice)
		penetration := distance.Div(bandWidth).InexactFloat64()
		strength = 0.6 + (penetration * 2) // Base strength + penetration factor
		if strength > 1.0 {
			strength = 1.0
		}
	} else if currentPrice.GreaterThan(currentUpper) {
		// Price is above upper band - potential sell signal (overbought)
		signal = "SELL"
		// Calculate how far above the upper band
		distance := currentPrice.Sub(currentUpper)
		penetration := distance.Div(bandWidth).InexactFloat64()
		strength = 0.6 + (penetration * 2)
		if strength > 1.0 {
			strength = 1.0
		}
	} else {
		// Price is within bands
		// Check for mean reversion opportunities
		upperThreshold := currentMiddle.Add(bandWidth.Mul(decimal.NewFromFloat(0.7)))
		lowerThreshold := currentMiddle.Sub(bandWidth.Mul(decimal.NewFromFloat(0.7)))

		if currentPrice.LessThan(lowerThreshold) {
			// Price is in lower 30% of band - weak buy signal
			signal = "BUY"
			distance := lowerThreshold.Sub(currentPrice)
			strength = 0.3 + (distance.Div(bandWidth).InexactFloat64() * 0.5)
		} else if currentPrice.GreaterThan(upperThreshold) {
			// Price is in upper 30% of band - weak sell signal
			signal = "SELL"
			distance := currentPrice.Sub(upperThreshold)
			strength = 0.3 + (distance.Div(bandWidth).InexactFloat64() * 0.5)
		} else {
			// No clear signal
			return nil
		}
	}

	// Additional validation: check recent price movement
	if len(prices) >= 3 {
		recentPrices := prices[len(prices)-3:]
		volatility := m.calculateVolatility(recentPrices)

		// Reduce strength if volatility is too high (risky conditions)
		if volatility > 0.05 { // 5% volatility threshold
			strength *= 0.7
		}
	}

	return &models.TradingSignal{
		Symbol:    symbol,
		Signal:    signal,
		Strength:  strength,
		Price:     currentPrice,
		Strategy:  m.GetName(),
		CreatedAt: time.Now(),
	}
}

// calculateVolatility calculates a simple volatility measure
func (m *MeanReversionStrategy) calculateVolatility(prices []decimal.Decimal) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	var totalVariance decimal.Decimal
	for i := 1; i < len(prices); i++ {
		change := prices[i].Sub(prices[i-1]).Div(prices[i-1])
		totalVariance = totalVariance.Add(change.Mul(change))
	}

	variance := totalVariance.Div(decimal.NewFromInt(int64(len(prices) - 1)))
	return variance.Pow(decimal.NewFromFloat(0.5)).InexactFloat64()
}
