package main

import (
	"bytes"
	"os"

	"github.com/golang/protobuf/jsonpb"
	inventorypb "github.com/netd-tud/ds-onlineshop/src/inventoryservice/genproto/inventory"
)

// loadInventory reads and parses product inventory data from a local JSON file
// into the provided inventory map.
func loadInventory(inventory map[string]*inventoryEntry) error {
	log.Info("loading inventory from local inventory.json file...")

	inventoryJSON, err := os.ReadFile("inventory.json")
	if err != nil {
		log.Warnf("failed to open product inventory json file: %v", err)
		return err
	}

	var resp inventorypb.ListInventoryResponse
	if err := jsonpb.Unmarshal(bytes.NewReader(inventoryJSON), &resp); err != nil {
		log.Warnf("failed to parse the inventory JSON: %v", err)
		return err
	}

	for _, p := range resp.Products {
		inventory[p.Id] = &inventoryEntry{
			product: p,
			owner:   systemUserIDSecret,
		}
	}

	log.Info("successfully parsed product inventory json")
	return nil
}
