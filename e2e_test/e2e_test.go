package e2e_test

import (
	"autonity-oralce/config"
	"autonity-oralce/http_server"
	"autonity-oralce/oracle_server"
	"autonity-oralce/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetVersion(t *testing.T) {
	conf := config.MakeConfig()
	// create oracle service and start the ticker job.
	oracle := oracle_server.NewOracleServer(conf.Symbols)
	go oracle.Start()
	defer oracle.Stop()

	// create http service.
	srv := http_server.NewHttpServer(oracle, conf.HttpPort)
	srv.StartHttpServer()

	// wait for the http service to be loaded.
	time.Sleep(5 * time.Second)

	var reqMsg = &types.JsonRpcMessage{
		Method: "get_version",
	}

	respMsg, err := httpPost(t, reqMsg, conf.HttpPort)
	require.NoError(t, err)
	type Version struct {
		Version string
	}
	var V Version
	err = json.Unmarshal(respMsg.Result, &V)
	require.NoError(t, err)
	require.Equal(t, oracle_server.Version, V.Version)

	defer srv.Shutdown(context.Background())
}

func TestGetPrices(t *testing.T) {
	conf := config.MakeConfig()
	// create oracle service and start the ticker job.
	oracle := oracle_server.NewOracleServer(conf.Symbols)
	go oracle.Start()
	defer oracle.Stop()

	// create http service.
	srv := http_server.NewHttpServer(oracle, conf.HttpPort)
	srv.StartHttpServer()

	// wait for oracle ticker job to fetch data from providers.
	time.Sleep(20 * time.Second)

	var reqMsg = &types.JsonRpcMessage{
		Method: "get_prices",
	}

	respMsg, err := httpPost(t, reqMsg, conf.HttpPort)
	require.NoError(t, err)
	type PriceAndSymbol struct {
		Prices  types.PriceBySymbol
		Symbols []string
	}
	var ps PriceAndSymbol
	err = json.Unmarshal(respMsg.Result, &ps)
	require.NoError(t, err)
	require.Equal(t, strings.Split(config.DefaultSymbols, ","), ps.Symbols)
	for _, s := range ps.Symbols {
		require.Equal(t, s, ps.Prices[s].Symbol)
		require.Equal(t, true, ps.Prices[s].Price.Equal(types.SimulatedPrice))
	}

	defer srv.Shutdown(context.Background())
}

func TestUpdateSymbols(t *testing.T) {
	conf := config.MakeConfig()
	// create oracle service and start the ticker job.
	oracle := oracle_server.NewOracleServer(conf.Symbols)
	go oracle.Start()
	defer oracle.Stop()

	// create http service.
	srv := http_server.NewHttpServer(oracle, conf.HttpPort)
	srv.StartHttpServer()

	// wait for http service to be ready.
	time.Sleep(5 * time.Second)

	newSymbols := []string{"NTNETH", "NTNBTC", "NTNBNB"}
	encSymbols, err := json.Marshal(newSymbols)
	require.NoError(t, err)

	var reqMsg = &types.JsonRpcMessage{
		Method: "update_symbols",
		Params: encSymbols,
	}

	respMsg, err := httpPost(t, reqMsg, conf.HttpPort)
	require.NoError(t, err)
	var symbols []string
	err = json.Unmarshal(respMsg.Result, &symbols)
	require.NoError(t, err)
	require.Equal(t, newSymbols, symbols)

	defer srv.Shutdown(context.Background())
}

func httpPost(t *testing.T, reqMsg *types.JsonRpcMessage, port int) (*types.JsonRpcMessage, error) {
	jsonData, err := json.Marshal(reqMsg)
	require.NoError(t, err)

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	var respMsg types.JsonRpcMessage
	err = json.NewDecoder(resp.Body).Decode(&respMsg)
	require.NoError(t, err)
	return &respMsg, nil
}
