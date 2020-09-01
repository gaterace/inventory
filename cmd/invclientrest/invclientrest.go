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
	"bytes"
	// "context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"regexp"
	"strconv"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
	"github.com/kylelemons/go-gypsy/yaml"

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
var json_data = flag.String("j", "", "json extension data")
var entity_name = flag.String("entity_name", "", "name of entity to be extended")

func main() {
	flag.Parse(true)

	configFilename := "conf.yaml"
	usr, err := user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		configFilename = homeDir + string(os.PathSeparator) + ".inventory.config"
	}

	config, err := yaml.ReadFile(configFilename)
	if err != nil {
		log.Fatalf("configuration not found: " + configFilename)
	}

	// log_file, _ := config.Get("log_file")
	ca_file, _ := config.Get("ca_file")
	useTls, _ := config.GetBool("tls")
	server_host_override, _ := config.Get("server_host_override")
	server, _ := config.Get("server")
	port, _ := config.GetInt("port")
	rest_port, _ := config.GetInt("rest_port")

	if port == 0 {
		port = 50055
	}

	if len(flag.Args()) < 1 {
		prog := os.Args[0]
		fmt.Printf("Command line client for inventory grpc service\n")
		fmt.Printf("usage:\n")
		fmt.Printf("    %s create_facility --name <name> [-j <json_data]\n", prog)
		fmt.Printf("    %s update_facility --id <facility_id> --version <version> --name <name> [-j <json_data]\n", prog)
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

		fmt.Printf("    %s create_subarea --facility <facility_id>  [--parent <subarea_id>] --position <position> --subtype <subarea_type_id> --name <name> [-j <json_data]\n", prog)
		fmt.Printf("    %s update_subarea --id <subarea_id>  [--parent <subarea_id>] --position <position> --subtype <subarea_type_id> --name <name> --version <version> [-j <json_data]\n", prog)
		fmt.Printf("    %s delete_subarea --id <subarea_id> --version <version>\n", prog)
		fmt.Printf("    %s get_subarea --id <subarea_id>\n", prog)
		fmt.Printf("    %s get_subareas --facility <facility_id>\n", prog)

		fmt.Printf("    %s create_product --name <name> [--sku <sku>] [--comment <comment>] [-j <json_data]\n", prog)
		fmt.Printf("    %s update_product --id <product_id> --name <name> [--sku <sku>] [--comment <comment>] --version <version> [-j <json_data]\n", prog)
		fmt.Printf("    %s delete_product --id <product_id> --version <version>\n", prog)
		fmt.Printf("    %s get_product --id <product_id>\n", prog)
		fmt.Printf("    %s get_products\n", prog)

		fmt.Printf("    %s create_item --subarea <subarea_id> --itemtype <item_type_id> --quantity <quantity> [--serial <serial_number>] --product <product_id> [-j <json_data]\n", prog)
		fmt.Printf("    %s update_item  --id <item_id> --version <version> --subarea <subarea_id> --itemtype <item_type_id> --quantity <quantity> [--serial <serial_number>] --product <product_id> [-j <json_data]\n", prog)
		fmt.Printf("    %s delete_item  --id <item_id> --version <version>\n", prog)
		fmt.Printf("    %s get_item  --id <item_id>\n", prog)
		fmt.Printf("    %s get_items_by_product --product <product_id>\n", prog)
		fmt.Printf("    %s get_items_by_subarea --subarea <subarea_id>\n", prog)
		fmt.Printf("    %s get_items_by_facility --facility <facility_id>\n", prog)

		fmt.Printf("    %s create_entity_schema --entity_name <entity_name> -j <json_schema> \n", prog)
		fmt.Printf("    %s update_entity_schema --entity_name <entity_name> -j <json_schema> \n", prog)
		fmt.Printf("    %s delete_entity_schema --entity_name <entity_name>\n", prog)
		fmt.Printf("    %s get_entity_schema --entity_name <entity_name>\n", prog)
		fmt.Printf("    %s get_entity_schemas\n", prog)


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
			fmt.Println("quantity parameter missing")
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

	case "create_entity_schema":

		if *entity_name == "" {
			fmt.Println("entity_name parameter missing")
			validParams = false
		}
		if *json_data == "" {
			fmt.Println("json_schema parameter missing")
			validParams = false
		}
	case "update_entity_schema":
		if *entity_name == "" {
			fmt.Println("entity_name parameter missing")
			validParams = false
		}
		if *json_data == "" {
			fmt.Println("json_schema parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "delete_entity_schema":
		if *entity_name == "" {
			fmt.Println("entity_name parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_entity_schema":
		if *entity_name == "" {
			fmt.Println("entity_name parameter missing")
			validParams = false
		}
	case "get_entity_schemas":
		// no parameters
		validParams = true
	case "get_server_version":
		// no parameters
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

	var serverAddr string

	if useTls {
		serverAddr = "https://" + server + ":" + strconv.Itoa(int(rest_port))
	} else {
		serverAddr = "http://" + server + ":" + strconv.Itoa(int(rest_port))
	}
	// fmt.Printf("address: %s\n", address)


	savedToken := ""

	data, err := ioutil.ReadFile(tokenFilename)

	if err == nil {
		savedToken = string(data)
	}



	bearer := "Bearer " + savedToken
	// fmt.Println(bearer)

	var client *http.Client

	if useTls {
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		if ca_file != "" {
			// Read in the cert file
			certs, err := ioutil.ReadFile(ca_file)
			if err != nil {
				log.Fatalf("Failed to append %q to RootCAs: %v", ca_file, err)
			}

			// Append our cert to the system pool
			if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
				log.Println("No certs appended, using system certs only")
			}
		}

		config := &tls.Config{}
		config.RootCAs = rootCAs
		config.ServerName = server_host_override

		tr := &http.Transport{TLSClientConfig: config}
		client = &http.Client{
			Transport: tr,
			Timeout: time.Second * 10,
		}


	} else {
		client = &http.Client{
			Timeout: time.Second * 10,
		}
	}


	switch cmd {
	case "create_facility":
		req := pb.CreateFacilityRequest{}
		req.FacilityName = *name
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := serverAddr + "/api/facility"
		doMuxRequest(url, bearer, client, "POST", json)


	case "update_facility":
		req := pb.UpdateFacilityRequest{}
		req.FacilityId = *id
		req.Version = int32(*version)
		req.FacilityName = *name
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := fmt.Sprintf("%s/api/facility/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "PUT", json)

	case "delete_facility":
		url := fmt.Sprintf("%s/api/facility/%d/%d", serverAddr, *id, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_facility":
		url := fmt.Sprintf("%s/api/facility/id/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_facilities":
		url := fmt.Sprintf("%s/api/facilities", serverAddr)
		doMuxRequest(url, bearer, client, "GET", nil)
	case "get_facility_wrapper":
		url := fmt.Sprintf("%s/api/facility/wrapper/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "create_subarea_type":
		req := pb.CreateSubareaTypeRequest{}
		req.SubareaTypeId = int32(*id)
		req.SubareaTypeName = *name
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := serverAddr + "/api/subareatype"
		doMuxRequest(url, bearer, client, "POST", json)

	case "update_subarea_type":
		req := pb.UpdateSubareaTypeRequest{}
		req.SubareaTypeId = int32(*id)
		req.SubareaTypeName = *name
		req.Version = int32(*version)
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := fmt.Sprintf("%s/api/subareatype/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "PUT", json)

	case "delete_subarea_type":
		url := fmt.Sprintf("%s/api/subareatype/%d/%d", serverAddr, *id, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_subarea_type":
		url := fmt.Sprintf("%s/api/subareatype/id/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_subarea_types":
		url := fmt.Sprintf("%s/api/subareatypes", serverAddr)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "create_item_type":
		req := pb.CreateItemTypeRequest{}
		req.ItemTypeId = int32(*id)
		req.ItemTypeName = *name
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := serverAddr + "/api/itemtype"
		doMuxRequest(url, bearer, client, "POST", json)

	case "update_item_type":
		req := pb.UpdateItemTypeRequest{}
		req.ItemTypeId = int32(*id)
		req.ItemTypeName = *name
		req.Version = int32(*version)
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := fmt.Sprintf("%s/api/itemtype/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "PUT", json)

	case "delete_item_type":
		url := fmt.Sprintf("%s/api/itemtype/%d/%d", serverAddr, *id, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_item_type":
		url := fmt.Sprintf("%s/api/itemtype/id/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_item_types":
		url := fmt.Sprintf("%s/api/itemtypes", serverAddr)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "create_subarea":
		req := pb.CreateSubareaRequest{}
		req.FacilityId = *facility
		req.ParentSubareaId = *parent
		req.Position = int32(*position)
		req.SubareaTypeId = int32(*subtype)
		req.SubareaName = *name
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := serverAddr + "/api/subarea"
		doMuxRequest(url, bearer, client, "POST", json)

	case "update_subarea":
		req := pb.UpdateSubareaRequest{}
		req.SubareaId = *id
		req.Version = int32(*version)
		req.ParentSubareaId = *parent
		req.Position = int32(*position)
		req.SubareaTypeId = int32(*subtype)
		req.SubareaName = *name
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := fmt.Sprintf("%s/api/subarea/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "PUT", json)

	case "delete_subarea":
		url := fmt.Sprintf("%s/api/subarea/%d/%d", serverAddr, *id, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_subarea":
		url := fmt.Sprintf("%s/api/subarea/id/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_subareas":
		url := fmt.Sprintf("%s/api/subareas/%d", serverAddr, *facility)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "create_product":
		req := pb.CreateProductRequest{}
		req.Sku = *sku
		req.ProductName = *name
		req.Comment = *comment
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := serverAddr + "/api/product"
		doMuxRequest(url, bearer, client, "POST", json)

	case "update_product":
		req := pb.UpdateProductRequest{}
		req.ProductId = *id
		req.Version = int32(*version)
		req.Sku = *sku
		req.ProductName = *name
		req.Comment = *comment
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := fmt.Sprintf("%s/api/product/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "PUT", json)


	case "delete_product":
		url := fmt.Sprintf("%s/api/product/%d/%d", serverAddr, *id, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_product":
		url := fmt.Sprintf("%s/api/product/id/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_products":
		url := fmt.Sprintf("%s/api/products", serverAddr)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "create_item":
		req := pb.CreateInventoryItemRequest{}
		req.SubareaId = *subarea
		req.ItemTypeId = int32(*itemtype)
		req.Quantity = int32(*quantity)
		req.SerialNumber = *serial
		req.ProductId = *product
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := serverAddr + "/api/item"
		doMuxRequest(url, bearer, client, "POST", json)

	case "update_item":
		req := pb.UpdateInventoryItemRequest{}
		req.InventoryItemId = *id
		req.Version = int32(*version)
		req.SubareaId = *subarea
		req.ItemTypeId = int32(*itemtype)
		req.Quantity = int32(*quantity)
		req.SerialNumber = *serial
		req.ProductId = *product
		req.JsonData = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}
		url := fmt.Sprintf("%s/api/item/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "PUT", json)


	case "delete_item":
		url := fmt.Sprintf("%s/api/item/%d/%d", serverAddr, *id, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_item":
		url := fmt.Sprintf("%s/api/item/id/%d", serverAddr, *id)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_items_by_product":
		url := fmt.Sprintf("%s/api/items/product/%d", serverAddr, *product)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_items_by_subarea":
		url := fmt.Sprintf("%s/api/items/subarea/%d", serverAddr, *subarea)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_items_by_facility":
		url := fmt.Sprintf("%s/api/items/facility/%d", serverAddr, *facility)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "create_entity_schema":
		req := pb.CreateEntitySchemaRequest{}
		req.EntityName = *entity_name
		req.JsonSchema = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}

		url := fmt.Sprintf("%s/api/schema", serverAddr)
		doMuxRequest(url, bearer, client, "POST", json)

	case "update_entity_schema":
		req := pb.UpdateEntitySchemaRequest{}
		req.EntityName = *entity_name
		req.Version = int32(*version)
		req.JsonSchema = *json_data
		json, err := requestToJson(req)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			break
		}

		url := fmt.Sprintf("%s/api/schema/%s", serverAddr, *entity_name)
		doMuxRequest(url, bearer, client, "PUT", json)

	case "delete_entity_schema":
		url := fmt.Sprintf("%s/api/schema/%s/%d", serverAddr, *entity_name, *version)
		doMuxRequest(url, bearer, client, "DELETE", nil)

	case "get_entity_schema":
		url := fmt.Sprintf("%s/api/schema/%s", serverAddr, *entity_name)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_entity_schemas":
		url := fmt.Sprintf("%s/api/schemas", serverAddr)
		doMuxRequest(url, bearer, client, "GET", nil)

	case "get_server_version":
		url := fmt.Sprintf("%s/api/server/version", serverAddr)
		doMuxRequest(url, bearer, client, "GET", nil)
	}

}

func doMuxRequest(url string, bearer string, client *http.Client, verb string, body io.Reader) {
	httpReq, err := http.NewRequest(verb, url, body)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		return
	}

	httpReq.Header.Set("Authorization", bearer)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(httpReq)
	if err == nil {
		respBody, _ :=  ioutil.ReadAll(resp.Body)
		fmt.Println(string(respBody))
	} else {
		fmt.Printf("err: %s\n", err)
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

func requestToJson(req interface{}) (*bytes.Buffer, error) {
	jtext, err := json.Marshal(req)
	// fmt.Printf("json: %s\n", string(jtext))
	buf := bytes.NewBuffer(jtext)
	return buf, err
}