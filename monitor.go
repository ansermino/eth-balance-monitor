package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"io"
	"math/big"
	"net/http"
	"path"
	"sync"
	"time"
)

type Monitor struct {
	ethUrl         string
	ethApi         *ethclient.Client
	updateInterval time.Duration
	httpPort       uint32
	accts          map[common.Address]*Account
	acctsLock      *sync.RWMutex
}

type Account struct {
	balance        *big.Int
	minimumBalance *big.Int
	healthy        bool
}

func NewMonitor(url string, port uint, accounts []common.Address, minimums []*big.Int, updateInterval int64) *Monitor {
	accts := make(map[common.Address]*Account, len(accounts))
	for i, a := range accounts {
		accts[a] = &Account{
			balance:        big.NewInt(0),
			minimumBalance: minimums[i],
			healthy:        true,
		}
	}

	app := &Monitor{
		ethUrl:         url,
		updateInterval: time.Second * time.Duration(updateInterval),
		httpPort:       uint32(port),
		accts:          accts,
		acctsLock:      &sync.RWMutex{},
	}

	return app
}

// Run starts monitoring the accounts and starts the http server
func (a *Monitor) Run() error {
	log.Info().
		Int64("interval", int64(a.updateInterval)).
		Str("url", a.ethUrl).
		Msg("Starting monitor")

	// Start rpc client
	rpcClient, err := ethclient.Dial(a.ethUrl)
	if err != nil {
		return err
	}
	a.ethApi = rpcClient

	// Log all the monitored accounts
	for addr, info := range a.accts {
		log.Info().
			Str("addr", addr.String()).
			Str("minimum", info.minimumBalance.String()).
			Msg("Monitoring account")
	}

	// Update the balances, then start serving requests
	updateBalances(a)
	go serveBalanceData(a)

	for {
		timeout := time.After(a.updateInterval)
		select {
		case <-timeout:
			updateBalances(a)
		}
	}
}

// updateBalances fetches balance info for all accounts and updates health status
func updateBalances(app *Monitor) {
	app.acctsLock.Lock()
	for addr, acct := range app.accts {
		bal, err := app.ethApi.BalanceAt(context.TODO(), addr, nil)
		if err != nil {
			log.Error().
				Err(err).
				Msgf("Failed to fetch balance for %s", addr.String())
			continue
		}

		// If balance is <= minimum balance
		if bal.Cmp(acct.minimumBalance) < 1 {
			log.Warn().
				Str("balance", bal.String()).
				Str("address", addr.String()).
				Msg("Account has low balance")
			acct.healthy = false
			acct.balance = bal
		} else {
			log.Info().
				Str("balance", bal.String()).
				Str("address", addr.String()).
				Msg("Account has a healthy balance")
			acct.healthy = true
			acct.balance = bal
		}
	}
	app.acctsLock.Unlock()
}

// Start serving account info on /balances/<addr>
func serveBalanceData(app *Monitor) {
	http.HandleFunc("/balances/", func(w http.ResponseWriter, r *http.Request) {
		handleBalanceRequest(app, w, r)
	})

	log.Fatal().Err(http.ListenAndServe(fmt.Sprintf(":%d", app.httpPort), nil))
}

func handleBalanceRequest(app *Monitor, w http.ResponseWriter, r *http.Request) {
	// Parse address from request URL and validate it
	base := path.Base(r.URL.String())
	if !common.IsHexAddress(base) {
		w.WriteHeader(404)
		_, _ = io.WriteString(w, "invalid address")
	}
	addr := common.HexToAddress(base)

	// Respond with status 200 if "healthy", otherwise 500. Both include balance as body.
	app.acctsLock.RLock()
	if info, ok := app.accts[addr]; ok {
		if info.healthy {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}
		_, _ = io.WriteString(w, info.balance.String())
	}
	app.acctsLock.RUnlock()
}
