package tests

import (
	"WalletAPI/m/internal/model"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func createWallet(t *testing.T) string {
	resp, err := httpClient.Post(baseURL+"/v1/create", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result model.Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.True(t, result.Success)

	data := result.Data.(map[string]interface{})
	walletID := data["walletId"].(string)
	require.NotEmpty(t, walletID)

	return walletID
}

func updateBalance(walletID, opType string, amount int64) (*http.Response, error) {
	reqBody := model.UpdateBalance{
		WalletId:      walletID,
		OperationType: opType,
		Amount:        amount,
	}

	body, _ := json.Marshal(reqBody)
	return httpClient.Post(baseURL+"/v1/wallet", "application/json", bytes.NewBuffer(body))
}

func getBalance(walletID string) (int64, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/v1/wallets/%s", baseURL, walletID))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result model.Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return 0, err
	}

	data := result.Data.(map[string]interface{})
	balance := int64(data["balance"].(float64))
	return balance, nil
}

func TestAPI_CreateWallet(t *testing.T) {
	walletID := createWallet(t)
	t.Logf("Created wallet: %s", walletID)
	assert.NotEmpty(t, walletID)
}

func TestAPI_DepositAndWithdraw(t *testing.T) {
	walletID := createWallet(t)

	resp, err := updateBalance(walletID, "DEPOSIT", 5000)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	balance, err := getBalance(walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), balance)

	resp, err = updateBalance(walletID, "WITHDRAW", 2000)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	balance, err = getBalance(walletID)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), balance)
}

// Тест: 1000 конкурентных запросов на один кошелек
func TestAPI_Concurrent_SingleWallet_1000Requests(t *testing.T) {
	walletID := createWallet(t)

	resp, err := updateBalance(walletID, "DEPOSIT", 1000000)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	const numRequests = 1000
	const amountPerOp = 100

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64
	var serverErrors atomic.Int64
	var statusCodes sync.Map

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			opType := "DEPOSIT"
			if index%2 == 0 {
				opType = "WITHDRAW"
			}

			resp, err := updateBalance(walletID, opType, amountPerOp)
			if err != nil {
				errorCount.Add(1)
				return
			}
			defer resp.Body.Close()

			statusCodes.Store(index, resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				successCount.Add(1)
			} else {
				errorCount.Add(1)
				if resp.StatusCode >= 500 {
					serverErrors.Add(1)
					t.Logf("ERROR: Request %d got 50x error: %d", index, resp.StatusCode)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	rps := float64(numRequests) / duration.Seconds()

	t.Logf("═══════════════════════════════════════")
	t.Logf("SINGLE WALLET - 1000 CONCURRENT REQUESTS")
	t.Logf("═══════════════════════════════════════")
	t.Logf("Total requests: %d", numRequests)
	t.Logf("Duration: %v", duration)
	t.Logf("RPS: %.2f", rps)
	t.Logf("Success: %d (%.2f%%)", successCount.Load(),
		float64(successCount.Load())/float64(numRequests)*100)
	t.Logf("Errors: %d", errorCount.Load())
	t.Logf("50x errors: %d", serverErrors.Load())
	t.Logf("═══════════════════════════════════════")

	assert.Equal(t, int64(0), serverErrors.Load(),
		"FAILED: Got %d server errors (50x). All requests must be processed!",
		serverErrors.Load())

	assert.Equal(t, int64(numRequests), successCount.Load(),
		"FAILED: Not all requests succeeded")

	time.Sleep(100 * time.Millisecond)
	finalBalance, err := getBalance(walletID)
	require.NoError(t, err)

	expectedBalance := int64(1000000)
	assert.Equal(t, expectedBalance, finalBalance,
		"Balance mismatch: expected %d, got %d", expectedBalance, finalBalance)
}

// Тест: несколько кошельков с конкурентными запросами
func TestAPI_Concurrent_MultipleWallets(t *testing.T) {
	const numWallets = 5
	const requestsPerWallet = 500
	const totalRequests = numWallets * requestsPerWallet

	wallets := make([]string, numWallets)
	for i := 0; i < numWallets; i++ {
		wallets[i] = createWallet(t)

		resp, err := updateBalance(wallets[i], "DEPOSIT", 100000)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		t.Logf("Created wallet %d: %s", i+1, wallets[i])
	}

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var serverErrors atomic.Int64

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			walletID := wallets[index%numWallets]

			opType := "DEPOSIT"
			if index%2 == 0 {
				opType = "WITHDRAW"
			}

			resp, err := updateBalance(walletID, opType, 50)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				successCount.Add(1)
			} else if resp.StatusCode >= 500 {
				serverErrors.Add(1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	rps := float64(totalRequests) / duration.Seconds()

	t.Logf("═══════════════════════════════════════")
	t.Logf("MULTIPLE WALLETS - %d WALLETS × %d REQUESTS", numWallets, requestsPerWallet)
	t.Logf("═══════════════════════════════════════")
	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Duration: %v", duration)
	t.Logf("RPS: %.2f", rps)
	t.Logf("Success: %d (%.2f%%)", successCount.Load(),
		float64(successCount.Load())/float64(totalRequests)*100)
	t.Logf("50x errors: %d", serverErrors.Load())
	t.Logf("═══════════════════════════════════════")

	assert.Equal(t, int64(0), serverErrors.Load(),
		"FAILED: Got %d server errors (50x)", serverErrors.Load())

	time.Sleep(100 * time.Millisecond)
	for i, walletID := range wallets {
		balance, err := getBalance(walletID)
		require.NoError(t, err)
		t.Logf("Wallet %d final balance: %d", i+1, balance)
		assert.Equal(t, int64(100000), balance)
	}
}

// Стресс-тест: 1000 RPS на протяжении 10 секунд
func TestAPI_Stress_1000RPS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	walletID := createWallet(t)

	resp, err := updateBalance(walletID, "DEPOSIT", 10000000)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	const targetRPS = 1000
	const testDuration = 10 * time.Second

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var serverErrors atomic.Int64
	var totalRequests atomic.Int64

	startTime := time.Now()
	endTime := startTime.Add(testDuration)

	ticker := time.NewTicker(time.Second / targetRPS)
	defer ticker.Stop()

	t.Logf("Starting stress test: target %d RPS for %v", targetRPS, testDuration)

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if time.Now().After(endTime) {
					return
				}

				wg.Add(1)
				totalRequests.Add(1)

				go func(count int64) {
					defer wg.Done()

					opType := "WITHDRAW"
					if count%3 == 0 {
						opType = "DEPOSIT"
					}

					resp, err := updateBalance(walletID, opType, 10)
					if err != nil {
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						successCount.Add(1)
					} else if resp.StatusCode >= 500 {
						serverErrors.Add(1)
					}
				}(totalRequests.Load())
			}
		}
	}()

	time.Sleep(testDuration)
	close(done)
	wg.Wait()

	actualDuration := time.Since(startTime)
	actualRPS := float64(totalRequests.Load()) / actualDuration.Seconds()

	t.Logf("═══════════════════════════════════════")
	t.Logf("STRESS TEST - 1000 RPS")
	t.Logf("═══════════════════════════════════════")
	t.Logf("Target RPS: %d", targetRPS)
	t.Logf("Actual RPS: %.2f", actualRPS)
	t.Logf("Duration: %v", actualDuration)
	t.Logf("Total requests: %d", totalRequests.Load())
	t.Logf("Success: %d (%.2f%%)", successCount.Load(),
		float64(successCount.Load())/float64(totalRequests.Load())*100)
	t.Logf("50x errors: %d", serverErrors.Load())
	t.Logf("═══════════════════════════════════════")

	assert.Equal(t, int64(0), serverErrors.Load(),
		"CRITICAL FAILURE: Got %d server errors (50x). System must handle all requests!",
		serverErrors.Load())

	successRate := float64(successCount.Load()) / float64(totalRequests.Load()) * 100
	assert.Greater(t, successRate, 99.0,
		"Success rate %.2f%% is below 99%%", successRate)
}
