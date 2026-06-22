package main

import (
	"bytes"
	"os"

	"github.com/golang/protobuf/jsonpb"
	pb "github.com/turt1z/microservices-demo/src/inventoryservice/genproto"
)

func loadInventory(inventory *pb.ListInventoryResponse) error {
	log.Info("loading inventory from local inventory.json file...")

	inventoryJSON, err := os.ReadFile("inventory.json")
	if err != nil {
		log.Warnf("failed to open product inventory json file: %v", err)
		return err
	}

	if err := jsonpb.Unmarshal(bytes.NewReader(inventoryJSON), inventory); err != nil {
		log.Warnf("failed to parse the inventory JSON: %v", err)
		return err
	}

	log.Info("successfully parsed product inventory json")
	return nil
}
