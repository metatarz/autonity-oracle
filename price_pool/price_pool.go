package price_pool

import (
	"autonity-oralce/types"
	"sync"
)

type PriceProvider struct {
	name          string
	lock          sync.RWMutex
	priceBySymbol types.PriceBySymbol
}

func NewPriceProvider(providerName string) *PriceProvider {
	return &PriceProvider{
		name:          providerName,
		priceBySymbol: make(types.PriceBySymbol),
	}
}

func (t *PriceProvider) AddPrices(prices []types.Price) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, p := range prices {
		t.priceBySymbol[p.Symbol] = p
	}
}

func (t *PriceProvider) GetPrice(symbol string) types.Price {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.priceBySymbol[symbol]
}

func (t *PriceProvider) GetPrices() types.PriceBySymbol {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.priceBySymbol
}

type PriceProviderPool struct {
	lock         sync.RWMutex
	providerPool map[string]*PriceProvider
}

func NewPriceProviderPool() *PriceProviderPool {
	return &PriceProviderPool{
		providerPool: make(map[string]*PriceProvider),
	}
}

func (tp *PriceProviderPool) AddPriceProvider(provider string) *PriceProvider {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	tp.providerPool[provider] = NewPriceProvider(provider)
	return tp.providerPool[provider]
}

func (tp *PriceProviderPool) GetPriceProvider(provider string) *PriceProvider {
	tp.lock.RLock()
	defer tp.lock.RUnlock()
	return tp.providerPool[provider]
}

func (tp *PriceProviderPool) DeletePriceProvider(provider string) {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	delete(tp.providerPool, provider)
}
