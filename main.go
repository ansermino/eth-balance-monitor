package main

import (
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math/big"
	"os"
	"strings"
)

/*
eth-balance-monitor is a simple service to enable monitoring of account balances.

A list of addresses and minimum balances are specified at runtime. The monitor regularly checks at intervals
the balance of each address. If a balance drops below the minimum it is marked as unhealthy.

An http server is started the provides endpoints for each address (eg. /balances/0x0123...). All responses have a
status code of 200 with the current balance as the body if the account is marked healthy, otherwise a 500 status code
is returned with the balance.
*/

// accountList is used to parse the addresses and minimums from a flag
type accountList struct {
	addrs    []common.Address
	minimums []*big.Int
}

func (al *accountList) String() string {
	return fmt.Sprintf("%v", *al)
}

// Set splits the input into addresses and minimum balances.
// The expected format is: <addresss>:<minimum>,....
func (al *accountList) Set(value string) error {
	list := strings.Split(value, ",")
	for _, acct := range list {
		tokens := strings.Split(acct, ":")

		if !common.IsHexAddress(tokens[0]) {
			return fmt.Errorf("invalid address: %s", tokens[0])
		}

		min, ok := new(big.Int).SetString(tokens[1], 10)
		if !ok {
			return fmt.Errorf("invalid minimum balance: %s", tokens[1])
		}

		al.addrs = append(al.addrs, common.HexToAddress(tokens[0]))
		al.minimums = append(al.minimums, min)
	}
	return nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var acctList accountList
	url := flag.String("url", "http://localhost:8545", "URL for ethereum node")
	port := flag.Uint("httpPort", 8080, "Port for http server")
	updateInterval := flag.Int64("interval", 60, "Seconds between balance updates")
	flag.Var(&acctList, "accounts", "Comma-separated list of accounts in the format: <address>:<minimum balance (wei)>")
	flag.Parse()

	if len(acctList.addrs) == 0 {
		log.Fatal().
			Msg("must provide -accounts")
	}

	app := NewMonitor(*url, *port, acctList.addrs, acctList.minimums, *updateInterval)
	err := app.Run()
	if err != nil {
		log.Error().
			Err(err)
	}
}
