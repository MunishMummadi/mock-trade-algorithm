package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/shopspring/decimal"

	"github.com/MunishMummadi/mock-trade-algorithm/alpaca"
	"github.com/MunishMummadi/mock-trade-algorithm/config"
	"github.com/MunishMummadi/mock-trade-algorithm/database"
	"github.com/MunishMummadi/mock-trade-algorithm/models"
	"github.com/MunishMummadi/mock-trade-algorithm/strategies"
)

type TradingEngine struct {
	config       *config.Config
	db           *database.Database
	alpacaClient *alpaca.Client
	strategies   []strategies.Strategy
	userID       int64
	running      bool
}

func main() {
	log.Println("Starting Mock Trade Algorithm...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Alpaca client
	alpacaClient, err := alpaca.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Alpaca client: %v", err)
	}

	// Create or get demo user
	user, err := getOrCreateDemoUser(db, cfg.InitialBalance)
	if err != nil {
		log.Fatalf("Failed to get or create demo user: %v", err)
	}

	log.Printf("Demo user created/found: %s (ID: %d, Balance: $%.2f)",
		user.Username, user.ID, user.Balance.InexactFloat64())

	// Initialize trading engine
	engine := &TradingEngine{
		config:       cfg,
		db:           db,
		alpacaClient: alpacaClient,
		userID:       user.ID,
		running:      true,
	}

	// Initialize trading strategies
	if err := engine.initializeStrategies(); err != nil {
		log.Fatalf("Failed to initialize trading strategies: %v", err)
	}

	// Start trading engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping trading engine...")
		engine.running = false
		cancel()
	}()

	// Start the trading loop
	if err := engine.run(ctx); err != nil {
		log.Fatalf("Trading engine error: %v", err)
	}

	log.Println("Mock Trade Algorithm stopped")
}

func getOrCreateDemoUser(db *database.Database, initialBalance float64) (*models.User, error) {
	// Try to get existing demo user
	user, err := db.GetUser(1)
	if err == nil {
		return user, nil
	}

	// Create new demo user
	user = models.NewUser(
		"demo_trader",
		"demo@example.com",
		decimal.NewFromFloat(initialBalance),
	)

	if err := db.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create demo user: %w", err)
	}

	return user, nil
}

func (e *TradingEngine) initializeStrategies() error {
	log.Println("Initializing trading strategies...")

	// Initialize Simple Moving Average strategy
	smaStrategy := strategies.NewSMAStrategy(20, 50) // 20-day and 50-day SMA
	e.strategies = append(e.strategies, smaStrategy)

	// Initialize RSI strategy
	rsiStrategy := strategies.NewRSIStrategy(14, 30, 70) // 14-period RSI with 30/70 levels
	e.strategies = append(e.strategies, rsiStrategy)

	// Initialize Mean Reversion strategy
	meanRevStrategy := strategies.NewMeanReversionStrategy(20, 2.0) // 20-period with 2 std dev
	e.strategies = append(e.strategies, meanRevStrategy)

	log.Printf("Initialized %d trading strategies", len(e.strategies))
	return nil
}

func (e *TradingEngine) run(ctx context.Context) error {
	log.Println("Starting trading engine main loop...")

	// Define watchlist of symbols to trade
	watchlist := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}

	ticker := time.NewTicker(e.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if !e.running {
				return nil
			}

			if err := e.processTradingCycle(ctx, watchlist); err != nil {
				log.Printf("Error in trading cycle: %v", err)
				continue
			}
		}
	}
}

func (e *TradingEngine) processTradingCycle(ctx context.Context, symbols []string) error {
	log.Println("Processing trading cycle...")

	// Check if market is open
	isOpen, err := e.alpacaClient.IsMarketOpen(ctx)
	if err != nil {
		return fmt.Errorf("failed to check market status: %w", err)
	}

	if !isOpen {
		log.Println("Market is closed, skipping trading cycle")
		return nil
	}

	// Get current prices for all symbols
	prices, err := e.alpacaClient.GetMultiplePrices(ctx, symbols)
	if err != nil {
		return fmt.Errorf("failed to get current prices: %w", err)
	}

	// Get current user and portfolio
	user, err := e.db.GetUser(e.userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	portfolio, err := e.db.GetPortfolioByUser(e.userID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Update portfolio values with current prices
	if err := e.updatePortfolioValues(portfolio, prices); err != nil {
		log.Printf("Warning: failed to update portfolio values: %v", err)
	}

	// Process each symbol with all strategies
	for _, symbol := range symbols {
		price, exists := prices[symbol]
		if !exists {
			log.Printf("Warning: price not available for %s", symbol)
			continue
		}

		if err := e.processSymbol(ctx, symbol, price, user, portfolio); err != nil {
			log.Printf("Error processing symbol %s: %v", symbol, err)
		}
	}

	// Print portfolio summary
	e.printPortfolioSummary(user, portfolio, prices)

	return nil
}

func (e *TradingEngine) processSymbol(ctx context.Context, symbol string, price decimal.Decimal,
	user *models.User, portfolio []*models.Portfolio) error {

	// Get historical data for analysis
	bars, err := e.alpacaClient.GetBars(ctx, symbol,
		marketdata.OneDay, time.Now().AddDate(0, 0, -100), time.Now())
	if err != nil {
		return fmt.Errorf("failed to get historical data for %s: %w", symbol, err)
	}

	if len(bars) < 50 { // Need enough data for analysis
		return nil
	}

	// Run all strategies for this symbol
	signals := make([]*models.TradingSignal, 0)

	for _, strategy := range e.strategies {
		signal := strategy.Analyze(symbol, bars, price)
		if signal != nil {
			signals = append(signals, signal)
		}
	}

	// Process signals and make trading decisions
	if len(signals) > 0 {
		decision := e.makeTradeDecision(signals, symbol, price, user, portfolio)
		if decision != nil {
			if err := e.executeTrade(ctx, decision, user); err != nil {
				log.Printf("Failed to execute trade for %s: %v", symbol, err)
			}
		}
	}

	return nil
}

func (e *TradingEngine) makeTradeDecision(signals []*models.TradingSignal, symbol string,
	currentPrice decimal.Decimal, user *models.User, portfolio []*models.Portfolio) *models.Trade {

	// Count buy and sell signals
	buySignals := 0
	sellSignals := 0
	totalStrength := 0.0

	for _, signal := range signals {
		if signal.Signal == "BUY" {
			buySignals++
		} else if signal.Signal == "SELL" {
			sellSignals++
		}
		totalStrength += signal.Strength
	}

	// Get current position for this symbol
	var currentPosition *models.Portfolio
	for _, pos := range portfolio {
		if pos.Symbol == symbol {
			currentPosition = pos
			break
		}
	}

	// Calculate position size based on risk management
	maxPositionValue := decimal.NewFromFloat(e.config.MaxPositionSize)
	riskAmount := user.Balance.Mul(decimal.NewFromFloat(e.config.RiskPercentage))

	// Decision logic
	if buySignals > sellSignals && buySignals >= 2 {
		// Strong buy signal
		if currentPosition == nil || currentPosition.Quantity.IsZero() {
			// Calculate quantity to buy
			positionValue := decimal.Min(maxPositionValue, riskAmount.Mul(decimal.NewFromFloat(totalStrength)))
			quantity := positionValue.Div(currentPrice).Truncate(0)

			if quantity.GreaterThan(decimal.Zero) && user.CanAfford(quantity.Mul(currentPrice)) {
				return models.NewTrade(user.ID, symbol, models.OrderSideBuy,
					models.TradeTypeMarket, quantity, currentPrice, "multi_strategy")
			}
		}
	} else if sellSignals > buySignals && sellSignals >= 2 {
		// Strong sell signal
		if currentPosition != nil && currentPosition.Quantity.GreaterThan(decimal.Zero) {
			// Sell the position
			return models.NewTrade(user.ID, symbol, models.OrderSideSell,
				models.TradeTypeMarket, currentPosition.Quantity, currentPrice, "multi_strategy")
		}
	}

	return nil
}

func (e *TradingEngine) executeTrade(ctx context.Context, trade *models.Trade, user *models.User) error {
	log.Printf("Executing %s trade: %s %s shares at $%.2f",
		trade.Side, trade.Quantity.String(), trade.Symbol, trade.Price.InexactFloat64())

	// Save trade to database
	if err := e.db.CreateTrade(trade); err != nil {
		return fmt.Errorf("failed to save trade: %w", err)
	}

	// Execute using mock trading (for safety)
	if err := e.alpacaClient.MockPlaceOrder(trade); err != nil {
		trade.Status = models.TradeStatusRejected
		e.db.UpdateTrade(trade)
		return fmt.Errorf("failed to execute trade: %w", err)
	}

	// Update trade status
	if err := e.db.UpdateTrade(trade); err != nil {
		return fmt.Errorf("failed to update trade: %w", err)
	}

	// Update user balance and portfolio
	if err := e.updateUserBalanceAndPortfolio(trade, user); err != nil {
		return fmt.Errorf("failed to update user balance and portfolio: %w", err)
	}

	log.Printf("Trade executed successfully: %s", trade.AlpacaOrderID)
	return nil
}

func (e *TradingEngine) updateUserBalanceAndPortfolio(trade *models.Trade, user *models.User) error {
	if trade.Status != models.TradeStatusFilled {
		return nil
	}

	// Calculate cost/proceeds
	totalCost := trade.GetTotalCost()

	if trade.Side == models.OrderSideBuy {
		user.UpdateBalance(totalCost.Neg())
	} else {
		user.UpdateBalance(totalCost)
	}

	// Update user in database
	if err := e.db.UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// Update portfolio
	portfolio := &models.Portfolio{
		UserID: user.ID,
		Symbol: trade.Symbol,
	}

	// Get existing position if any
	existingPortfolio, err := e.db.GetPortfolioByUser(user.ID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	for _, p := range existingPortfolio {
		if p.Symbol == trade.Symbol {
			portfolio = p
			break
		}
	}

	// Update position
	quantity := trade.Quantity
	if trade.Side == models.OrderSideSell {
		quantity = quantity.Neg()
	}

	portfolio.UpdatePosition(quantity, trade.FillPrice)

	// Save portfolio
	if err := e.db.UpsertPortfolio(portfolio); err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	return nil
}

func (e *TradingEngine) updatePortfolioValues(portfolio []*models.Portfolio,
	prices map[string]decimal.Decimal) error {

	for _, position := range portfolio {
		if price, exists := prices[position.Symbol]; exists {
			position.CalculateUnrealizedPL(price)
			if err := e.db.UpsertPortfolio(position); err != nil {
				return fmt.Errorf("failed to update portfolio position: %w", err)
			}
		}
	}

	return nil
}

func (e *TradingEngine) printPortfolioSummary(user *models.User, portfolio []*models.Portfolio,
	prices map[string]decimal.Decimal) {

	log.Println("=== Portfolio Summary ===")
	log.Printf("Cash Balance: $%.2f", user.Balance.InexactFloat64())

	totalValue := user.Balance
	totalPL := decimal.Zero

	for _, position := range portfolio {
		if !position.Quantity.IsZero() {
			currentPrice := prices[position.Symbol]
			log.Printf("%s: %s shares @ $%.2f (avg: $%.2f) = $%.2f (P&L: $%.2f)",
				position.Symbol,
				position.Quantity.String(),
				currentPrice.InexactFloat64(),
				position.AveragePrice.InexactFloat64(),
				position.CurrentValue.InexactFloat64(),
				position.UnrealizedPL.InexactFloat64())

			totalValue = totalValue.Add(position.CurrentValue)
			totalPL = totalPL.Add(position.UnrealizedPL)
		}
	}

	log.Printf("Total Portfolio Value: $%.2f", totalValue.InexactFloat64())
	log.Printf("Total Unrealized P&L: $%.2f", totalPL.InexactFloat64())
	log.Println("========================")
}
