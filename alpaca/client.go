package alpaca

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/config"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
)

type Client struct {
	alpacaClient *alpaca.Client
	marketClient *marketdata.Client
	config       *config.Config
}

func NewClient(cfg *config.Config) (*Client, error) {
	// Initialize Alpaca trading client
	alpacaClient := alpaca.NewClient(alpaca.ClientOpts{
		APIKey:    cfg.AlpacaAPIKey,
		APISecret: cfg.AlpacaAPISecret,
		BaseURL:   cfg.AlpacaBaseURL,
	})

	// Initialize market data client
	marketClient := marketdata.NewClient(marketdata.ClientOpts{
		APIKey:    cfg.AlpacaAPIKey,
		APISecret: cfg.AlpacaAPISecret,
	})

	client := &Client{
		alpacaClient: alpacaClient,
		marketClient: marketClient,
		config:       cfg,
	}

	// Verify connection
	if err := client.verifyConnection(); err != nil {
		return nil, fmt.Errorf("failed to verify Alpaca connection: %w", err)
	}

	log.Println("Successfully connected to Alpaca API")
	return client, nil
}

func (c *Client) verifyConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection by getting account information
	_, err := c.alpacaClient.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	return nil
}

func (c *Client) GetAccount(ctx context.Context) (*alpaca.Account, error) {
	return c.alpacaClient.GetAccount(ctx)
}

func (c *Client) PlaceOrder(ctx context.Context, trade *models.Trade) (*alpaca.Order, error) {
	orderSide := alpaca.Buy
	if trade.Side == models.OrderSideSell {
		orderSide = alpaca.Sell
	}

	orderType := alpaca.Market
	switch trade.Type {
	case models.TradeTypeLimit:
		orderType = alpaca.Limit
	case models.TradeTypeStop:
		orderType = alpaca.Stop
	}

	orderRequest := alpaca.PlaceOrderRequest{
		Symbol:      trade.Symbol,
		Qty:         trade.Quantity,
		Side:        orderSide,
		Type:        orderType,
		TimeInForce: alpaca.DAY,
	}

	if trade.Type == models.TradeTypeLimit || trade.Type == models.TradeTypeStop {
		orderRequest.LimitPrice = &trade.Price
	}

	order, err := c.alpacaClient.PlaceOrder(ctx, orderRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	return order, nil
}

func (c *Client) GetOrder(ctx context.Context, orderID string) (*alpaca.Order, error) {
	return c.alpacaClient.GetOrder(ctx, orderID)
}

func (c *Client) CancelOrder(ctx context.Context, orderID string) error {
	return c.alpacaClient.CancelOrder(ctx, orderID)
}

func (c *Client) GetPositions(ctx context.Context) ([]alpaca.Position, error) {
	return c.alpacaClient.GetPositions(ctx)
}

func (c *Client) GetPosition(ctx context.Context, symbol string) (*alpaca.Position, error) {
	return c.alpacaClient.GetPosition(ctx, symbol)
}

func (c *Client) GetLatestQuote(ctx context.Context, symbol string) (*marketdata.LatestQuote, error) {
	quotes, err := c.marketClient.GetLatestQuotes(ctx, marketdata.GetLatestQuotesRequest{
		Symbols: []string{symbol},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get latest quote for %s: %w", symbol, err)
	}

	quote, exists := quotes[symbol]
	if !exists {
		return nil, fmt.Errorf("quote not found for symbol %s", symbol)
	}

	return &quote, nil
}

func (c *Client) GetLatestTrade(ctx context.Context, symbol string) (*marketdata.LatestTrade, error) {
	trades, err := c.marketClient.GetLatestTrades(ctx, marketdata.GetLatestTradesRequest{
		Symbols: []string{symbol},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get latest trade for %s: %w", symbol, err)
	}

	trade, exists := trades[symbol]
	if !exists {
		return nil, fmt.Errorf("trade not found for symbol %s", symbol)
	}

	return &trade, nil
}

func (c *Client) GetBars(ctx context.Context, symbol string, timeframe marketdata.TimeFrame,
	start, end time.Time) ([]marketdata.Bar, error) {

	req := marketdata.GetBarsRequest{
		Symbols:   []string{symbol},
		TimeFrame: timeframe,
		Start:     start,
		End:       end,
	}

	resp, err := c.marketClient.GetBars(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get bars for %s: %w", symbol, err)
	}

	bars, exists := resp[symbol]
	if !exists {
		return nil, fmt.Errorf("bars not found for symbol %s", symbol)
	}

	return bars, nil
}

func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	quote, err := c.GetLatestQuote(ctx, symbol)
	if err != nil {
		// Fallback to latest trade if quote is not available
		trade, tradeErr := c.GetLatestTrade(ctx, symbol)
		if tradeErr != nil {
			return decimal.Zero, fmt.Errorf("failed to get current price for %s: quote error: %w, trade error: %w",
				symbol, err, tradeErr)
		}
		return decimal.NewFromFloat(trade.Price), nil
	}

	// Use mid-price between bid and ask
	bid := decimal.NewFromFloat(quote.BidPrice)
	ask := decimal.NewFromFloat(quote.AskPrice)
	return bid.Add(ask).Div(decimal.NewFromInt(2)), nil
}

func (c *Client) GetMultiplePrices(ctx context.Context, symbols []string) (map[string]decimal.Decimal, error) {
	quotes, err := c.marketClient.GetLatestQuotes(ctx, marketdata.GetLatestQuotesRequest{
		Symbols: symbols,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get quotes for symbols: %w", err)
	}

	prices := make(map[string]decimal.Decimal)
	for symbol, quote := range quotes {
		bid := decimal.NewFromFloat(quote.BidPrice)
		ask := decimal.NewFromFloat(quote.AskPrice)
		prices[symbol] = bid.Add(ask).Div(decimal.NewFromInt(2))
	}

	// For symbols without quotes, try to get trades
	for _, symbol := range symbols {
		if _, exists := prices[symbol]; !exists {
			trade, err := c.GetLatestTrade(ctx, symbol)
			if err != nil {
				log.Printf("Warning: failed to get price for %s: %v", symbol, err)
				continue
			}
			prices[symbol] = decimal.NewFromFloat(trade.Price)
		}
	}

	return prices, nil
}

func (c *Client) IsMarketOpen(ctx context.Context) (bool, error) {
	clock, err := c.alpacaClient.GetClock(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get market clock: %w", err)
	}

	return clock.IsOpen, nil
}

func (c *Client) GetMarketCalendar(ctx context.Context, start, end time.Time) ([]alpaca.CalendarDay, error) {
	return c.alpacaClient.GetCalendar(ctx, &alpaca.GetCalendarRequest{
		Start: &start,
		End:   &end,
	})
}

func (c *Client) StreamTrades(ctx context.Context, symbols []string, handler func(marketdata.Trade)) error {
	tradeHandler := func(trade marketdata.Trade) {
		handler(trade)
	}

	return c.marketClient.SubscribeTrades(ctx, tradeHandler, symbols...)
}

func (c *Client) StreamQuotes(ctx context.Context, symbols []string, handler func(marketdata.Quote)) error {
	quoteHandler := func(quote marketdata.Quote) {
		handler(quote)
	}

	return c.marketClient.SubscribeQuotes(ctx, quoteHandler, symbols...)
}

// Mock trading methods for paper trading
func (c *Client) MockPlaceOrder(trade *models.Trade) error {
	// Simulate order processing delay
	time.Sleep(100 * time.Millisecond)

	// For mock trading, we'll simulate immediate fills at current market price
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	currentPrice, err := c.GetCurrentPrice(ctx, trade.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get current price for mock order: %w", err)
	}

	// Add some realistic slippage for market orders
	slippage := decimal.NewFromFloat(0.001) // 0.1% slippage
	if trade.Type == models.TradeTypeMarket {
		if trade.Side == models.OrderSideBuy {
			currentPrice = currentPrice.Mul(decimal.NewFromInt(1).Add(slippage))
		} else {
			currentPrice = currentPrice.Mul(decimal.NewFromInt(1).Sub(slippage))
		}
	}

	// Mock commission (Alpaca has commission-free trading, but we can simulate)
	commission := decimal.Zero

	trade.MarkFilled(currentPrice, commission)
	trade.AlpacaOrderID = fmt.Sprintf("mock_%d_%s", time.Now().Unix(), trade.Symbol)

	return nil
}
