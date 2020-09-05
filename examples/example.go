package main

import (
	"log"

	"github.com/onflow/cadence"
	"github.com/versus-flow/go-flow-tooling/tooling"
)

func main() {

	gwtf := tooling.NewGoWithTheFlowEmulator()

	gwtf.DeployContract("nft")
	gwtf.DeployContract("ft")
	gwtf.SendTransaction("create_nft_collection", "ft")
	gwtf.SendTransactionWithArguments("arguments", "ft", cadence.String("argument1"))

	//create an argument that is a cadence.Address from the wallet.json file
	gwtf.SendTransactionWithArguments("argumentsWithAccount", "ft", gwtf.FindAddress("nft"))

	// Run Script
	gwtf.RunScriptReturns("test", gwtf.FindAddress("nft"))

	//Run script that returns
	result := gwtf.RunScriptReturns("test", gwtf.FindAddress("nft"))
	log.Printf("Script returned %s", result)

}
