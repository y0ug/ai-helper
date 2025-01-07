package cost

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	pricingURL     = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"
	pricingFile    = "model_prices.json"
	cacheDuration  = 24 * time.Hour
)

type Tracker struct {
	pricing    PricingData
	lastUpdate time.Time
	mu         sync.RWMutex
}

func NewTracker() (*Tracker, error) {
	configDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir = filepath.Join(configDir, ".config", "ai-helper")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	t := &Tracker{}
	if err := t.loadPricing(); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tracker) loadPricing() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	configDir, _ := os.UserHomeDir()
	pricingPath := filepath.Join(configDir, ".config", "ai-helper", pricingFile)

	// Check if we need to update the pricing file
	needsUpdate := true
	if info, err := os.Stat(pricingPath); err == nil {
		t.lastUpdate = info.ModTime()
		if time.Since(t.lastUpdate) < cacheDuration {
			needsUpdate = false
		}
	}

	if needsUpdate {
		if err := t.downloadPricing(pricingPath); err != nil {
			return fmt.Errorf("failed to download pricing: %w", err)
		}
	}

	data, err := os.ReadFile(pricingPath)
	if err != nil {
		return fmt.Errorf("failed to read pricing file: %w", err)
	}

	if err := json.Unmarshal(data, &t.pricing); err != nil {
		return fmt.Errorf("failed to parse pricing data: %w", err)
	}

	return nil
}

func (t *Tracker) downloadPricing(pricingPath string) error {
	resp, err := http.Get(pricingURL)
	if err != nil {
		return fmt.Errorf("failed to download pricing data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download pricing data: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read pricing data: %w", err)
	}

	if err := os.WriteFile(pricingPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write pricing file: %w", err)
	}

	return nil
}

func (t *Tracker) CalculateCost(model string, inputTokens, outputTokens int) (float64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	pricing, err := t.pricing.GetModelPricing(model)
	if err != nil {
		return 0, err
	}

	inputCost := float64(inputTokens) * pricing.GetInputCostPerToken()
	outputCost := float64(outputTokens) * pricing.GetOutputCostPerToken()

	return inputCost + outputCost, nil
}
