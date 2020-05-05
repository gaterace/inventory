// Copyright 2019 Demian Harvill
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command line GRPC client for MServiceInventory.
package main

import (
	"context"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"regexp"
	"strconv"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
	"github.com/kylelemons/go-gypsy/yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"

	flag "github.com/juju/gnuflag"
)

var idlistValidator = regexp.MustCompile("^\\d+(,\\d+)*$")

var name = flag.String("name", "", "name")
var id = flag.Int64("id", -1, "id")
var version = flag.Int("version", -1, "version")
var subtype = flag.Int("subtype", -1, "subarea type id")
var itemtype = flag.Int("itemtype", -1, "item type id")
var facility = flag.Int64("facility", -1, "facility_id")
var parent = flag.Int64("parent", 0, "parent_id")
var position = flag.Int("position", -1, "position")
var sku = flag.String("sku", "", "sku")
var comment = flag.String("comment", "", "comment")
var subarea = flag.Int64("subarea", -1, "subarea_id")
var quantity = flag.Int("quantity", -1, "quantity")
var serial = flag.String("serial", "", "serial")
var product = flag.Int64("product", -1, "product_id")

func main() {
	flag.Parse(true)

	configFilename := "conf.yaml"
	usr, err := user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		configFilename = homeDir + string(os.PathSeparator) + ".inventory.config"
		// _ = homeDir + string(os.PathSeparator) + ".inventory.config"
	}

	config, err := yaml.ReadFile(configFilename)
	if err != nil {
		log.Fatalf("configuration not found: " + configFilename)
	}

	// log_file, _ := config.Get("log_file")
	ca_file, _ := config.Get("ca_file")
	tls, _ := config.GetBool("tls")
	server_host_override, _ := config.Get("server_host_override")
	server, _ := config.Get("server")
	port, _ := config.GetInt("port")

	if port == 0 {
		port = 50055
	}

	if len(flag.Args()) < 1 {
		prog := os.Args[0]
		fmt.Printf("Command line client for inventory grpc service\n")
		fmt.Printf("usage:\n")
		fmt.Printf("    %s create_facility --name <name>\n", prog)
		fmt.Printf("    %s update_facility --id <facility_id> --version <version> --name <name>\n", prog)
		fmt.Printf("    %s delete_facility --id <facility_id> --version <version>\n", prog)
		fmt.Printf("    %s get_facility --id <facility_id>\n", prog)
		fmt.Printf("    %s get_facilities\n", prog)
		fmt.Printf("    %s get_facility_wrapper --id <facility_id>\n", prog)

		fmt.Printf("    %s create_subarea_type  --id <subarea_type_id> --name <name>\n", prog)
		fmt.Printf("    %s update_subarea_type  --id <subarea_type_id> --name <name> --version <version>\n", prog)
		fmt.Printf("    %s delete_subarea_type  --id <subarea_type_id> --version <version>\n", prog)
		fmt.Printf("    %s get_subarea_type  --id <subarea_type_id>\n", prog)
		fmt.Printf("    %s get_subarea_types\n", prog)

		fmt.Printf("    %s create_item_type  --id <item_type_id> --name <name>\n", prog)
		fmt.Printf("    %s update_item_type  --id <item_type_id> --name <name> --version <version>\n", prog)
		fmt.Printf("    %s delete_item_type  --id <item_type_id> --version <version>\n", prog)
		fmt.Printf("    %s get_item_type  --id <item_type_id>\n", prog)
		fmt.Printf("    %s get_item_types\n", prog)

		fmt.Printf("    %s create_subarea --facility <facility_id>  [--parent <subarea_id>] --position <position> --subtype <subarea_type_id> --name <name>\n", prog)
		fmt.Printf("    %s update_subarea --id <subarea_id>  [--parent <subarea_id>] --position <position> --subtype <subarea_type_id> --name <name> --version <version>\n", prog)
		fmt.Printf("    %s delete_subarea --id <subarea_id> --version <version>\n", prog)
		fmt.Printf("    %s get_subarea --id <subarea_id>\n", prog)
		fmt.Printf("    %s get_subareas --facility <facility_id>\n", prog)

		fmt.Printf("    %s create_product --name <name> [--sku <sku>] [--comment <comment>]\n", prog)
		fmt.Printf("    %s update_product --id <product_id> --name <name> [--sku <sku>] [--comment <comment>] --version <version>\n", prog)
		fmt.Printf("    %s delete_product --id <product_id> --version <version>\n", prog)
		fmt.Printf("    %s get_product --id <product_id>\n", prog)
		fmt.Printf("    %s get_products\n", prog)

		fmt.Printf("    %s create_item --subarea <subarea_id> --itemtype <item_type_id> --quantity <quantity> [--serial <serial_number>] --product <product_id>\n", prog)
		fmt.Printf("    %s update_item  --id <item_id> --version <version> --subarea <subarea_id> --itemtype <item_type_id> --quantity <quantity> [--serial <serial_number>] --product <product_id>\n", prog)
		fmt.Printf("    %s delete_item  --id <item_id> --version <version>\n", prog)
		fmt.Printf("    %s get_item  --id <item_id>\n", prog)
		fmt.Printf("    %s get_items_by_product --product <product_id>\n", prog)
		fmt.Printf("    %s get_items_by_subarea --subarea <subarea_id>\n", prog)
		fmt.Printf("    %s get_items_by_facility --facility <facility_id>\n", prog)

		fmt.Printf("    %s get_server_version\n", prog)
	}

	cmd := flag.Arg(0)

	validParams := true

	switch cmd {
	case "create_facility":

		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
	case "update_facility":
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "delete_facility":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_facility":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}

	case "get_facilities":
		// no params
		validParams = true

	case "get_facility_wrapper":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}

	case "create_subarea_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

	case "update_subarea_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "delete_subarea_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_subarea_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}

	case "get_subarea_types":
		// no params
		validParams = true

	case "create_item_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

	case "update_item_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "delete_item_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_item_type":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}

	case "get_item_types":
		// no parameters
		validParams = true

	case "create_subarea":
		if *facility == -1 {
			fmt.Println("facility parameter missing")
			validParams = false
		}

		if *position == -1 {
			fmt.Println("position parameter missing")
			validParams = false
		}
		if *subtype == -1 {
			fmt.Println("subtype parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

	case "update_subarea":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *position == -1 {
			fmt.Println("position parameter missing")
			validParams = false
		}
		if *subtype == -1 {
			fmt.Println("subtype parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "delete_subarea":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_subarea":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}

	case "get_subareas":
		if *facility == -1 {
			fmt.Println("facility parameter missing")
			validParams = false
		}

	case "create_product":
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
	case "update_product":
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "delete_product":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_product":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
	case "get_products":
		// no parameters
		validParams = true

	case "create_item":
		if *subarea == -1 {
			fmt.Println("subarea parameter missing")
			validParams = false
		}
		if *itemtype == -1 {
			fmt.Println("itemtype parameter missing")
			validParams = false
		}
		if *quantity == -1 {
			fmt.Println("itemtype parameter missing")
			validParams = false
		}
		if *product == -1 {
			fmt.Println("product parameter missing")
			validParams = false
		}

	case "update_item":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
		if *subarea == -1 {
			fmt.Println("subarea parameter missing")
			validParams = false
		}
		if *itemtype == -1 {
			fmt.Println("itemtype parameter missing")
			validParams = false
		}
		if *quantity == -1 {
			fmt.Println("itemtype parameter missing")
			validParams = false
		}
		if *product == -1 {
			fmt.Println("product parameter missing")
			validParams = false
		}
	case "delete_item":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_item":
		if *id == -1 {
			fmt.Println("id parameter missing")
			validParams = false
		}
	case "get_items_by_product":
		if *product == -1 {
			fmt.Println("product parameter missing")
			validParams = false
		}
	case "get_items_by_subarea":
		if *subarea == -1 {
			fmt.Println("subarea parameter missing")
			validParams = false
		}
	case "get_items_by_facility":
		if *facility == -1 {
			fmt.Println("facility parameter missing")
			validParams = false
		}

	case "get_server_version":
		// no paramaters
		validParams = true

	default:
		fmt.Printf("unknown command: %s\n", cmd)
		validParams = false
	}

	if !validParams {
		os.Exit(1)
	}

	tokenFilename := "token.txt"
	usr, err = user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		tokenFilename = homeDir + string(os.PathSeparator) + ".mservice.token"
	}

	address := server + ":" + strconv.Itoa(int(port))
	// fmt.Printf("address: %s\n", address)

	var opts []grpc.DialOption
	if tls {
		var sn string
		if server_host_override != "" {
			sn = server_host_override
		}
		var creds credentials.TransportCredentials
		if ca_file != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(ca_file, sn)
			if err != nil {
				grpclog.Fatalf("Failed to create TLS credentials %v", err)
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, sn)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// set up connection to server
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	client := pb.NewMServiceInventoryClient(conn)

	ctx := context.Background()

	savedToken := ""

	data, err := ioutil.ReadFile(tokenFilename)

	if err == nil {
		savedToken = string(data)
	}

	md := metadata.Pairs("token", savedToken)
	mctx := metadata.NewOutgoingContext(ctx, md)

	switch cmd {
	case "create_facility":
		req := pb.CreateFacilityRequest{}
		req.FacilityName = *name
		resp, err := client.CreateFacility(mctx, &req)
		printResponse(resp, err)

	case "update_facility":
		req := pb.UpdateFacilityRequest{}
		req.FacilityId = *id
		req.Version = int32(*version)
		req.FacilityName = *name
		resp, err := client.UpdateFacility(mctx, &req)
		printResponse(resp, err)

	case "delete_facility":
		req := pb.DeleteFacilityRequest{}
		req.FacilityId = *id
		req.Version = int32(*version)
		resp, err := client.DeleteFacility(mctx, &req)
		printResponse(resp, err)

	case "get_facility":
		req := pb.GetFacilityRequest{}
		req.FacilityId = *id
		resp, err := client.GetFacility(mctx, &req)
		printResponse(resp, err)

	case "get_facilities":
		req := pb.GetFacilitiesRequest{}
		resp, err := client.GetFacilities(mctx, &req)
		printResponse(resp, err)

	case "get_facility_wrapper":
		req := pb.GetFacilityWrapperRequest{}
		req.FacilityId = *id
		resp, err := client.GetFacilityWrapper(mctx, &req)
		printResponse(resp, err)

	case "create_subarea_type":
		req := pb.CreateSubareaTypeRequest{}
		req.SubareaTypeId = int32(*id)
		req.SubareaTypeName = *name
		resp, err := client.CreateSubareaType(mctx, &req)
		printResponse(resp, err)

	case "update_subarea_type":
		req := pb.UpdateSubareaTypeRequest{}
		req.SubareaTypeId = int32(*id)
		req.SubareaTypeName = *name
		req.Version = int32(*version)
		resp, err := client.UpdateSubareaType(mctx, &req)
		printResponse(resp, err)

	case "delete_subarea_type":
		req := pb.DeleteSubareaTypeRequest{}
		req.SubareaTypeId = int32(*id)
		req.Version = int32(*version)
		resp, err := client.DeleteSubareaType(mctx, &req)
		printResponse(resp, err)

	case "get_subarea_type":
		req := pb.GetSubareaTypeRequest{}
		req.SubareaTypeId = int32(*id)
		resp, err := client.GetSubareaType(mctx, &req)
		printResponse(resp, err)

	case "get_subarea_types":
		req := pb.GetSubareaTypesRequest{}
		resp, err := client.GetSubareaTypes(mctx, &req)
		printResponse(resp, err)

	case "create_item_type":
		req := pb.CreateItemTypeRequest{}
		req.ItemTypeId = int32(*id)
		req.ItemTypeName = *name
		resp, err := client.CreateItemType(mctx, &req)
		printResponse(resp, err)

	case "update_item_type":
		req := pb.UpdateItemTypeRequest{}
		req.ItemTypeId = int32(*id)
		req.ItemTypeName = *name
		req.Version = int32(*version)
		resp, err := client.UpdateItemType(mctx, &req)
		printResponse(resp, err)

	case "delete_item_type":
		req := pb.DeleteItemTypeRequest{}
		req.ItemTypeId = int32(*id)
		req.Version = int32(*version)
		resp, err := client.DeleteItemType(mctx, &req)
		printResponse(resp, err)

	case "get_item_type":
		req := pb.GetItemTypeRequest{}
		req.ItemTypeId = int32(*id)
		resp, err := client.GetItemType(mctx, &req)
		printResponse(resp, err)

	case "get_item_types":
		req := pb.GetItemTypesRequest{}
		resp, err := client.GetItemTypes(mctx, &req)
		printResponse(resp, err)

	case "create_subarea":
		req := pb.CreateSubareaRequest{}
		req.FacilityId = *facility
		req.ParentSubareaId = *parent
		req.Position = int32(*position)
		req.SubareaTypeId = int32(*subtype)
		req.SubareaName = *name
		resp, err := client.CreateSubarea(mctx, &req)
		printResponse(resp, err)

	case "update_subarea":
		req := pb.UpdateSubareaRequest{}
		req.SubareaId = *id
		req.Version = int32(*version)
		req.ParentSubareaId = *parent
		req.Position = int32(*position)
		req.SubareaTypeId = int32(*subtype)
		req.SubareaName = *name
		resp, err := client.UpdateSubarea(mctx, &req)
		printResponse(resp, err)

	case "delete_subarea":
		req := pb.DeleteSubareaRequest{}
		req.SubareaId = *id
		req.Version = int32(*version)
		resp, err := client.DeleteSubarea(mctx, &req)
		printResponse(resp, err)

	case "get_subarea":
		req := pb.GetSubareaRequest{}
		req.SubareaId = *id
		resp, err := client.GetSubarea(mctx, &req)
		printResponse(resp, err)

	case "get_subareas":
		req := pb.GetSubareasRequest{}
		req.FacilityId = *facility
		resp, err := client.GetSubareas(mctx, &req)
		printResponse(resp, err)

	case "create_product":
		req := pb.CreateProductRequest{}
		req.Sku = *sku
		req.ProductName = *name
		req.Comment = *comment
		resp, err := client.CreateProduct(mctx, &req)
		printResponse(resp, err)

	case "update_product":
		req := pb.UpdateProductRequest{}
		req.ProductId = *id
		req.Version = int32(*version)
		req.Sku = *sku
		req.ProductName = *name
		req.Comment = *comment
		resp, err := client.UpdateProduct(mctx, &req)
		printResponse(resp, err)

	case "delete_product":
		req := pb.DeleteProductRequest{}
		req.ProductId = *id
		req.Version = int32(*version)
		resp, err := client.DeleteProduct(mctx, &req)
		printResponse(resp, err)

	case "get_product":
		req := pb.GetProductRequest{}
		req.ProductId = *id
		resp, err := client.GetProduct(mctx, &req)
		printResponse(resp, err)

	case "get_products":
		req := pb.GetProductsRequest{}
		resp, err := client.GetProducts(mctx, &req)
		printResponse(resp, err)

	case "create_item":
		req := pb.CreateInventoryItemRequest{}
		req.SubareaId = *subarea
		req.ItemTypeId = int32(*itemtype)
		req.Quantity = int32(*quantity)
		req.SerialNumber = *serial
		req.ProductId = *product
		resp, err := client.CreateInventoryItem(mctx, &req)
		printResponse(resp, err)

	case "update_item":
		req := pb.UpdateInventoryItemRequest{}
		req.InventoryItemId = *id
		req.Version = int32(*version)
		req.SubareaId = *subarea
		req.ItemTypeId = int32(*itemtype)
		req.Quantity = int32(*quantity)
		req.SerialNumber = *serial
		req.ProductId = *product
		resp, err := client.UpdateInventoryItem(mctx, &req)
		printResponse(resp, err)

	case "delete_item":
		req := pb.DeleteInventoryItemRequest{}
		req.InventoryItemId = *id
		req.Version = int32(*version)
		resp, err := client.DeleteInventoryItem(mctx, &req)
		printResponse(resp, err)

	case "get_item":
		req := pb.GetInventoryItemRequest{}
		req.InventoryItemId = *id
		resp, err := client.GetInventoryItem(mctx, &req)
		printResponse(resp, err)

	case "get_items_by_product":
		req := pb.GetInventoryItemsByProductRequest{}
		req.ProductId = *product
		resp, err := client.GetInventoryItemsByProduct(mctx, &req)
		printResponse(resp, err)

	case "get_items_by_subarea":
		req := pb.GetInventoryItemsBySubareaRequest{}
		req.SubareaId = *subarea
		resp, err := client.GetInventoryItemsBySubarea(mctx, &req)
		printResponse(resp, err)

	case "get_items_by_facility":
		req := pb.GetInventoryItemsByFacilityRequest{}
		req.FacilityId = *facility
		resp, err := client.GetInventoryItemsByFacility(mctx, &req)
		printResponse(resp, err)

	case "get_server_version":
		req := pb.GetServerVersionRequest{}
		resp, err := client.GetServerVersion(mctx, &req)
		printResponse(resp, err)

	}

}

// Helper to print method response as JSON.
func printResponse(resp interface{}, err error) {
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
}
