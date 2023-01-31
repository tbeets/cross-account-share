package main

import (
	"os"

	"cross-account-sourcing/poc"
)

func main() {
	wd, _ := os.Getwd()
	rf := poc.StartRetailFleet(wd)
	defer poc.StopRetailFleet(rf)

	select {}
}
