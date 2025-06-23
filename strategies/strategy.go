package strategies

import (
	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/alpaca"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

// Strategy interface that all trading strategies must implement
type Strategy interface {
	// Analyze takes historical data and current price, returns a trading signal
	Analyze(symbol string, bars []alpaca.MockBar, currentPrice decimal.Decimal) *models.TradingSignal

	// GetName returns the strategy name
	GetName() string

	// GetDescription returns a description of the strategy
	GetDescription() string
}

// BaseStrategy provides common functionality for all strategies
type BaseStrategy struct {
	name        string
	description string
}

func (bs *BaseStrategy) GetName() string {
	return bs.name
}

func (bs *BaseStrategy) GetDescription() string {
	return bs.description
}

// Helper functions for technical analysis

// CalculateSMA calculates Simple Moving Average
func CalculateSMA(values []decimal.Decimal, period int) []decimal.Decimal {
	if len(values) < period {
		return nil
	}

	sma := make([]decimal.Decimal, len(values)-period+1)

	for i := period - 1; i < len(values); i++ {
		sum := decimal.Zero
		for j := i - period + 1; j <= i; j++ {
			sum = sum.Add(values[j])
		}
		sma[i-period+1] = sum.Div(decimal.NewFromInt(int64(period)))
	}

	return sma
}

// CalculateEMA calculates Exponential Moving Average
func CalculateEMA(values []decimal.Decimal, period int) []decimal.Decimal {
	if len(values) < period {
		return nil
	}

	ema := make([]decimal.Decimal, len(values))
	multiplier := decimal.NewFromFloat(2.0).Div(decimal.NewFromInt(int64(period + 1)))

	// Start with SMA for the first value
	sum := decimal.Zero
	for i := 0; i < period; i++ {
		sum = sum.Add(values[i])
	}
	ema[period-1] = sum.Div(decimal.NewFromInt(int64(period)))

	// Calculate EMA for remaining values
	for i := period; i < len(values); i++ {
		ema[i] = values[i].Mul(multiplier).Add(ema[i-1].Mul(decimal.NewFromInt(1).Sub(multiplier)))
	}

	return ema[period-1:]
}

// CalculateRSI calculates Relative Strength Index
func CalculateRSI(prices []decimal.Decimal, period int) []decimal.Decimal {
	if len(prices) < period+1 {
		return nil
	}

	gains := make([]decimal.Decimal, len(prices)-1)
	losses := make([]decimal.Decimal, len(prices)-1)

	// Calculate price changes
	for i := 1; i < len(prices); i++ {
		change := prices[i].Sub(prices[i-1])
		if change.IsPositive() {
			gains[i-1] = change
			losses[i-1] = decimal.Zero
		} else {
			gains[i-1] = decimal.Zero
			losses[i-1] = change.Abs()
		}
	}

	// Calculate initial averages
	avgGain := decimal.Zero
	avgLoss := decimal.Zero

	for i := 0; i < period; i++ {
		avgGain = avgGain.Add(gains[i])
		avgLoss = avgLoss.Add(losses[i])
	}

	avgGain = avgGain.Div(decimal.NewFromInt(int64(period)))
	avgLoss = avgLoss.Div(decimal.NewFromInt(int64(period)))

	rsi := make([]decimal.Decimal, len(gains)-period+1)

	// Calculate RSI values
	for i := period - 1; i < len(gains); i++ {
		if i > period-1 {
			// Smoothed averages
			avgGain = avgGain.Mul(decimal.NewFromInt(int64(period - 1))).Add(gains[i]).Div(decimal.NewFromInt(int64(period)))
			avgLoss = avgLoss.Mul(decimal.NewFromInt(int64(period - 1))).Add(losses[i]).Div(decimal.NewFromInt(int64(period)))
		}

		if avgLoss.IsZero() {
			rsi[i-period+1] = decimal.NewFromInt(100)
		} else {
			rs := avgGain.Div(avgLoss)
			rsi[i-period+1] = decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
		}
	}

	return rsi
}

// CalculateBollingerBands calculates Bollinger Bands
func CalculateBollingerBands(prices []decimal.Decimal, period int, stdDev float64) ([]decimal.Decimal, []decimal.Decimal, []decimal.Decimal) {
	if len(prices) < period {
		return nil, nil, nil
	}

	sma := CalculateSMA(prices, period)
	if sma == nil {
		return nil, nil, nil
	}

	upper := make([]decimal.Decimal, len(sma))
	lower := make([]decimal.Decimal, len(sma))
	middle := sma

	for i := 0; i < len(sma); i++ {
		// Calculate standard deviation for this period
		sum := decimal.Zero
		startIdx := i + period - 1

		for j := startIdx - period + 1; j <= startIdx; j++ {
			diff := prices[j].Sub(sma[i])
			sum = sum.Add(diff.Mul(diff))
		}

		variance := sum.Div(decimal.NewFromInt(int64(period)))
		stdDevValue := decimal.NewFromFloat(stdDev).Mul(variance.Pow(decimal.NewFromFloat(0.5)))

		upper[i] = sma[i].Add(stdDevValue)
		lower[i] = sma[i].Sub(stdDevValue)
	}

	return upper, middle, lower
}

// CalculateMACD calculates Moving Average Convergence Divergence
func CalculateMACD(prices []decimal.Decimal, fastPeriod, slowPeriod, signalPeriod int) ([]decimal.Decimal, []decimal.Decimal, []decimal.Decimal) {
	if len(prices) < slowPeriod {
		return nil, nil, nil
	}

	fastEMA := CalculateEMA(prices, fastPeriod)
	slowEMA := CalculateEMA(prices, slowPeriod)

	if len(fastEMA) == 0 || len(slowEMA) == 0 {
		return nil, nil, nil
	}

	// Align the EMAs (slow EMA starts later)
	alignedLen := len(slowEMA)
	fastAligned := fastEMA[len(fastEMA)-alignedLen:]

	// Calculate MACD line
	macdLine := make([]decimal.Decimal, alignedLen)
	for i := 0; i < alignedLen; i++ {
		macdLine[i] = fastAligned[i].Sub(slowEMA[i])
	}

	// Calculate signal line (EMA of MACD line)
	signalLine := CalculateEMA(macdLine, signalPeriod)

	// Calculate histogram
	histogramLen := len(signalLine)
	histogram := make([]decimal.Decimal, histogramLen)
	macdAligned := macdLine[len(macdLine)-histogramLen:]

	for i := 0; i < histogramLen; i++ {
		histogram[i] = macdAligned[i].Sub(signalLine[i])
	}

	return macdAligned, signalLine, histogram
}

// ExtractPrices extracts closing prices from bars
func ExtractPrices(bars []alpaca.MockBar) []decimal.Decimal {
	prices := make([]decimal.Decimal, len(bars))
	for i, bar := range bars {
		prices[i] = decimal.NewFromFloat(bar.Close)
	}
	return prices
}

// ExtractHighs extracts high prices from bars
func ExtractHighs(bars []alpaca.MockBar) []decimal.Decimal {
	highs := make([]decimal.Decimal, len(bars))
	for i, bar := range bars {
		highs[i] = decimal.NewFromFloat(bar.High)
	}
	return highs
}

// ExtractLows extracts low prices from bars
func ExtractLows(bars []alpaca.MockBar) []decimal.Decimal {
	lows := make([]decimal.Decimal, len(bars))
	for i, bar := range bars {
		lows[i] = decimal.NewFromFloat(bar.Low)
	}
	return lows
}
