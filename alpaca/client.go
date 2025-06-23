package alpaca

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/config"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

type Client struct {
	config       *config.Config
	mockPrices   map[string]decimal.Decimal
	mockAccounts map[string]decimal.Decimal
}

type MockAccount struct {
	ID            string          `json:"id"`
	AccountNumber string          `json:"account_number"`
	Status        string          `json:"status"`
	Cash          decimal.Decimal `json:"cash"`
	BuyingPower   decimal.Decimal `json:"buying_power"`
}

type MockPosition struct {
	Symbol        string          `json:"symbol"`
	Qty           decimal.Decimal `json:"qty"`
	AvgEntryPrice decimal.Decimal `json:"avg_entry_price"`
	MarketValue   decimal.Decimal `json:"market_value"`
}

type MockOrder struct {
	ID        string          `json:"id"`
	Symbol    string          `json:"symbol"`
	Qty       decimal.Decimal `json:"qty"`
	Side      string          `json:"side"`
	OrderType string          `json:"order_type"`
	Status    string          `json:"status"`
	Price     decimal.Decimal `json:"price"`
	CreatedAt time.Time       `json:"created_at"`
}

type MockBar struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
}

func NewClient(cfg *config.Config) (*Client, error) {
	client := &Client{
		config:       cfg,
		mockPrices:   make(map[string]decimal.Decimal),
		mockAccounts: make(map[string]decimal.Decimal),
	}

	// Initialize mock prices for common stocks
	client.initializeMockPrices()

	log.Println("Successfully initialized Mock Alpaca API client")
	return client, nil
}

func (c *Client) initializeMockPrices() {
	// Initialize with realistic stock prices
	stockPrices := map[string]float64{
		"AAPL":  175.50,
		"GOOGL": 135.25,
		"MSFT":  378.85,
		"TSLA":  238.45,
		"AMZN":  145.30,
		"NVDA":  875.25,
		"META":  485.60,
		"NFLX":  425.75,
	}

	for symbol, price := range stockPrices {
		c.mockPrices[symbol] = decimal.NewFromFloat(price)
	}
}

func (c *Client) GetAccount(ctx context.Context) (*MockAccount, error) {
	return &MockAccount{
		ID:            "mock_account_123",
		AccountNumber: "123456789",
		Status:        "ACTIVE",
		Cash:          decimal.NewFromFloat(c.config.InitialBalance),
		BuyingPower:   decimal.NewFromFloat(c.config.InitialBalance * 2), // 2x leverage
	}, nil
}

func (c *Client) IsMarketOpen(ctx context.Context) (bool, error) {
	now := time.Now()
	// Simple mock: market is open Monday-Friday 9:30 AM - 4:00 PM ET
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false, nil
	}

	hour := now.Hour()
	return hour >= 9 && hour < 16, nil
}

func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	// Get base price
	basePrice, exists := c.mockPrices[symbol]
	if !exists {
		return decimal.Zero, fmt.Errorf("price not available for symbol %s", symbol)
	}

	// Add some random fluctuation (±2%)
	fluctuation := (rand.Float64() - 0.5) * 0.04 // -2% to +2%
	currentPrice := basePrice.Mul(decimal.NewFromFloat(1 + fluctuation))

	// Update the stored price for next time
	c.mockPrices[symbol] = currentPrice

	return currentPrice, nil
}

func (c *Client) GetMultiplePrices(ctx context.Context, symbols []string) (map[string]decimal.Decimal, error) {
	prices := make(map[string]decimal.Decimal)

	for _, symbol := range symbols {
		price, err := c.GetCurrentPrice(ctx, symbol)
		if err != nil {
			log.Printf("Warning: failed to get price for %s: %v", symbol, err)
			continue
		}
		prices[symbol] = price
	}

	return prices, nil
}

func (c *Client) GetBars(ctx context.Context, symbol string, timeframe interface{}, start, end time.Time) ([]MockBar, error) {
	// Generate mock historical data
	var bars []MockBar

	basePrice, exists := c.mockPrices[symbol]
	if !exists {
		basePrice = decimal.NewFromFloat(100.0) // Default price
	}

	current := start
	price := basePrice.InexactFloat64()

	for current.Before(end) && len(bars) < 100 {
		// Generate realistic OHLC data
		open := price

		// Random daily change (±5%)
		change := (rand.Float64() - 0.5) * 0.1
		close := open * (1 + change)

		// High and low based on volatility
		volatility := rand.Float64() * 0.03 // 0-3% intraday range
		high := open * (1 + volatility)
		low := open * (1 - volatility)

		// Ensure OHLC makes sense
		if close > high {
			high = close
		}
		if close < low {
			low = close
		}

		bars = append(bars, MockBar{
			Timestamp: current,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    int64(rand.Intn(10000000) + 1000000), // 1M-11M volume
		})

		price = close
		current = current.AddDate(0, 0, 1) // Next day
	}

	return bars, nil
}

func (c *Client) MockPlaceOrder(trade *models.Trade) error {
	// Simulate order processing delay
	time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)

	// Get current price for the symbol
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	currentPrice, err := c.GetCurrentPrice(ctx, trade.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get current price for mock order: %w", err)
	}

	// Simulate market impact and slippage
	slippage := c.calculateSlippage(trade)
	fillPrice := currentPrice

	if trade.Type == models.TradeTypeMarket {
		if trade.Side == models.OrderSideBuy {
			fillPrice = currentPrice.Add(slippage)
		} else {
			fillPrice = currentPrice.Sub(slippage)
		}
	} else {
		// For limit orders, fill at the limit price if market allows
		if trade.Side == models.OrderSideBuy && currentPrice.LessThanOrEqual(trade.Price) {
			fillPrice = trade.Price
		} else if trade.Side == models.OrderSideSell && currentPrice.GreaterThanOrEqual(trade.Price) {
			fillPrice = trade.Price
		} else {
			// Order doesn't fill immediately
			trade.Status = models.TradeStatusPending
			trade.AlpacaOrderID = fmt.Sprintf("mock_pending_%d_%s", time.Now().Unix(), trade.Symbol)
			return nil
		}
	}

	// Mock commission (Alpaca is commission-free, but we can simulate other costs)
	commission := decimal.Zero

	// Random chance of rejection (1% for realism)
	if rand.Float64() < 0.01 {
		trade.Status = models.TradeStatusRejected
		trade.AlpacaOrderID = fmt.Sprintf("mock_rejected_%d_%s", time.Now().Unix(), trade.Symbol)
		return fmt.Errorf("mock order rejected due to insufficient funds or market conditions")
	}

	trade.MarkFilled(fillPrice, commission)
	trade.AlpacaOrderID = fmt.Sprintf("mock_%d_%s", time.Now().Unix(), trade.Symbol)

	log.Printf("Mock order filled: %s %s %s @ $%.2f",
		trade.Side, trade.Quantity.String(), trade.Symbol, fillPrice.InexactFloat64())

	return nil
}

func (c *Client) calculateSlippage(trade *models.Trade) decimal.Decimal {
	// Calculate slippage based on order size and market conditions
	baseSlippage := decimal.NewFromFloat(0.001) // 0.1% base slippage

	// Larger orders have more slippage
	sizeMultiplier := trade.Quantity.Div(decimal.NewFromInt(100))
	if sizeMultiplier.GreaterThan(decimal.NewFromInt(1)) {
		baseSlippage = baseSlippage.Mul(sizeMultiplier)
	}

	// Add some randomness
	randomFactor := decimal.NewFromFloat(rand.Float64() * 0.002) // 0-0.2% random
	slippage := baseSlippage.Add(randomFactor)

	// Apply to current price
	currentPrice := c.mockPrices[trade.Symbol]
	return currentPrice.Mul(slippage)
}

// Simplified methods that don't rely on complex external APIs
func (c *Client) GetOrder(ctx context.Context, orderID string) (*MockOrder, error) {
	// Mock implementation
	return &MockOrder{
		ID:        orderID,
		Symbol:    "AAPL",
		Qty:       decimal.NewFromInt(10),
		Side:      "buy",
		OrderType: "market",
		Status:    "filled",
		Price:     decimal.NewFromFloat(175.50),
		CreatedAt: time.Now(),
	}, nil
}

func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	// Mock implementation
	log.Printf("Mock: Cancelled order %s", orderID)
	return nil
}

func (c *Client) GetPositions(ctx context.Context) ([]MockPosition, error) {
	// Mock implementation - return empty positions
	return []MockPosition{}, nil
}

func (c *Client) GetPosition(ctx context.Context, symbol string) (*MockPosition, error) {
	// Mock implementation
	return &MockPosition{
		Symbol:        symbol,
		Qty:           decimal.Zero,
		AvgEntryPrice: decimal.Zero,
		MarketValue:   decimal.Zero,
	}, nil
}
