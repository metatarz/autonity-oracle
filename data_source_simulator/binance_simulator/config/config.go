package config

import (
	"github.com/namsral/flag"
	"github.com/shopspring/decimal"
	"strings"
)

var (
	// DefSimulatorConf contains a list of data items with the pattern of SYMBOL:StartingDataPoint:DataDistributionRateRange
	// with each separated by a "|". To make things be simple, we assume that the default opening rate of NTNUSD is
	// 1.0, and the other pair's opening rate are pre-computed by fiat money exchange rate. Thus, the starting data point
	// for each symbol list here also reflect the exchange rate between the corresponding fiat money, to tune the
	// starting data point for symbols, one can set the values on demand with CLI flags or system environment variables
	// with such data pattern, or just use the magnification factor parameter to increase or decrease the starting data point.
	DefSimulatorConf                            = "NTNUSD:1.0:0.01|NTNAUD:1.408:0.01|NTNCAD:1.3333:0.01|NTNEUR:0.9767:0.01|NTNGBP:0.813:0.01|NTNJPY:128.205:0.01|NTNSEK:10.309:0.01"
	DefSimulatorPort                            = 50991 // default port bind with the http service in the simulator.
	DefDataPointMagnificationFactor             = 7.0   // the default starting points are multiplied by this factor for increasing or decreasing.
	DefDataDistributionRangeMagnificationFactor = 2.0   // the default data distribution rate range is multiplied by this factor for increasing or decreasing.
)

type GeneratorConfig struct {
	ReferenceDataPoint decimal.Decimal
	DistributionRate   decimal.Decimal
}

type SimulatorConfig struct {
	Port          int
	SimulatorConf map[string]*GeneratorConfig
}

func MakeSimulatorConfig() *SimulatorConfig {
	var port int
	var simulatorConf string

	flag.IntVar(&port, "sim_http_port", DefSimulatorPort, "The HTTP rpc port to be bind for binance_simulator simulator")
	flag.StringVar(&simulatorConf, "sim_symbol_config", DefSimulatorConf,
		"The list of data items with the pattern of SYMBOL:StartingDataPoint:DataDistributionRateRange with each separated by a \"|\"")
	dataPointMagnificationFactor := flag.Float64("sim_data_magnification_factor", DefDataPointMagnificationFactor,
		"The magnification factor to increase or decrease symbols' starting data point")
	distributionRangeMagnificaitonFactor := flag.Float64("sim_data_dist_range_magnification_factor",
		DefDataDistributionRangeMagnificationFactor, "The magnification factor to increase or decrease the range of the rate for random data distribution")

	flag.Parse()

	conf := ParseSimulatorConf(simulatorConf, decimal.NewFromFloat(*dataPointMagnificationFactor),
		decimal.NewFromFloat(*distributionRangeMagnificaitonFactor))

	return &SimulatorConfig{
		Port:          port,
		SimulatorConf: conf,
	}
}

func ParseSimulatorConf(conf string, dataPointFactor decimal.Decimal, distributionRateFactor decimal.Decimal) map[string]*GeneratorConfig {
	println("\n\n\n\tRunning simulator with conf: ", conf)
	println("\twith data point factor: ", dataPointFactor.String())
	println("\twith data distribution rate factor: ", distributionRateFactor.String())

	result := make(map[string]*GeneratorConfig)
	items := strings.Split(conf, "|")
	for _, it := range items {
		i := strings.TrimSpace(it)
		if len(i) == 0 {
			continue
		}
		fields := strings.Split(i, ":")
		if len(fields) != 3 {
			continue
		}

		symbol := fields[0]
		startPoint := fields[1]
		rateRange := fields[2]
		result[symbol] = &GeneratorConfig{
			ReferenceDataPoint: decimal.RequireFromString(startPoint).Mul(dataPointFactor),
			DistributionRate:   decimal.RequireFromString(rateRange).Mul(distributionRateFactor),
		}
	}
	return result
}
