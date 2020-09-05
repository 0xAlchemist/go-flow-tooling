package tooling

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// RawFlowConfig for marshalling into simple types
type RawFlowConfig struct {
	Address          string
	GasLimit         uint64
	Accounts         map[string]RawAccount
	EmulatorAccounts map[string]string
}

// RawAccount flow accounts struct for marshalling into primitive types
type RawAccount struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	SigAlgo    string `json:"sigAlgorithm"`
	HashAlgo   string `json:"hashAlgorithm"`
}

// NewRawFlowConfig will read the flow.json file
func NewRawFlowConfig(path string) (*RawFlowConfig, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read flow json file")
	}

	d := json.NewDecoder(f)

	var flowConfig RawFlowConfig
	err = d.Decode(&flowConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Could not decode json info RawFlowCOnfig")
	}

	return &flowConfig, nil
}
