package autonity_oralce

import (
	"autonity-oralce/aggregator"
	"autonity-oralce/price_pool"
	"autonity-oralce/provider/crypto_provider"
	"autonity-oralce/types"
	"github.com/shopspring/decimal"
	"sync"
	"time"
)

var PERIOD = 3 * 60 * 1000

type OracleService struct {
	version string
	config  *types.OracleServiceConfig

	lock sync.RWMutex
	// aggregated prices
	prices types.PriceBySymbol

	doneCh chan struct{}
	ticker *time.Ticker

	symbols           []string
	aggregator        types.Aggregator
	priceProviderPool *price_pool.PriceProviderPool
	adapters          []types.Adapter
}

func NewOracleService(config *types.OracleServiceConfig) *OracleService {
	os := &OracleService{
		version:           "v0.0.1",
		config:            config,
		symbols:           config.Symbols,
		prices:            make(types.PriceBySymbol),
		doneCh:            make(chan struct{}),
		ticker:            time.NewTicker(10 * time.Second),
		aggregator:        aggregator.NewAveragePriceAggregator(),
		priceProviderPool: price_pool.NewPriceProviderPool(),
	}

	for _, provider := range config.Providers {
		if provider == "Binance" {
			pool := os.priceProviderPool.AddPriceProvider(provider)
			adapter := crypto_provider.NewBinanceAdapter()
			os.adapters = append(os.adapters, adapter)
			adapter.Initialize(pool)
		} else {
			continue
		}
	}
	return os
}

func (os *OracleService) Version() string {
	return os.version
}

func (os *OracleService) UpdateSymbols(symbols []string) {
	os.symbols = symbols
}

func (os *OracleService) Symbols() []string {
	return os.symbols
}

func (os *OracleService) GetPrice(symbol string) types.Price {
	os.lock.RLock()
	defer os.lock.RUnlock()
	return os.prices[symbol]
}

func (os *OracleService) GetPrices() types.PriceBySymbol {
	os.lock.RLock()
	defer os.lock.RUnlock()
	return os.prices
}

func (os *OracleService) UpdatePrice(symbol string, price types.Price) {
	os.lock.Lock()
	defer os.lock.Unlock()
	os.prices[symbol] = price
}

func (os *OracleService) UpdatePrices() {
	// todo: launch multiple go routine fetch price for all symbols from all adaptors.
	for _, ad := range os.adapters {
		ad.FetchPrices(os.symbols)
	}

	now := time.Now().UnixMilli()

	for _, s := range os.symbols {
		var prices []decimal.Decimal
		for _, ad := range os.adapters {
			p := os.priceProviderPool.GetPriceProvider(ad.Name()).GetPrice(s)
			// only those price collected within 3 minutes are valid.
			if now-p.Timestamp < int64(PERIOD) && now >= p.Timestamp {
				prices = append(prices, p.Price)
			}
		}

		if len(prices) == 0 {
			continue
		}

		price := types.Price{
			Timestamp: now,
			Price:     prices[0],
			Symbol:    s,
		}

		if len(prices) > 1 {
			p := os.aggregator.Aggregate(prices)
			price.Price = p
		}

		os.UpdatePrice(s, price)
	}
}

func (os *OracleService) Stop() {
	os.doneCh <- struct{}{}
}

func (os *OracleService) Start() {
	// start ticker job.
	for {
		select {
		case <-os.doneCh:
			os.ticker.Stop()
			return
		case <-os.ticker.C:
			os.UpdatePrices()
		}
	}
}
