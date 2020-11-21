# eth-balance-monitor

eth-balance-monitor is a simple service to enable monitoring of account balances.

A list of addresses and minimum balances are specified at runtime. The monitor regularly checks at intervals
the balance of each address. If a balance drops below the minimum it is marked as unhealthy.

An http server is started the provides endpoints for each address (eg. /balances/0x0123...). All responses have a
status code of 200 with the current balance as the body if the account is marked healthy, otherwise a 500 status code
is returned with the balance.

## Usage

```
eth-balance-monitor -url http://localhost:8545 -httpPort 8080 -interval 60 -accounts 0xff93B45308FD417dF303D6515aB04D9e89a750Ca:100000,0x8e0a907331554AF72563Bd8D43051C2E64Be5d35:1000000
```

You can query one this balances with:
```
curl http://localhost:8080/balances/0xff93B45308FD417dF303D6515aB04D9e89a750Ca
```