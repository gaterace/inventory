// Copyright 2019-2020 Demian Harvill
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

// Package muxhandler provides the gorilla mux handler for mservice rest.

package muxhandler

import (
	"context"
	"encoding/json"
	"github.com/gaterace/inventory/pkg/invauth"
	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"strconv"

	// "io/ioutil"
	"net/http"
	// "strconv"
)

type muxHandler struct {
	auth *invauth.InvAuth
	rtr *mux.Router
}

// Create aa new muxHandler struct
func NewMuxHandler(invauth *invauth.InvAuth, rtr *mux.Router) *muxHandler {
	mh := muxHandler{}
	mh.auth = invauth
	mh.rtr = rtr

	return &mh
}

// Add the gorilla mux handlers.
func (mh *muxHandler) AddRoutes() {
	// TODO
	mh.rtr.HandleFunc("/api/facility", mh.CreateFacilityHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/facility/{id:[0-9]+}", mh.UpdateFacilityHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/facility/{id:[0-9]+}/{version:[0-9]+}", mh.DeleteFacilityHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/facility/id/{id:[0-9]+}", mh.GetFacilityHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/facilities", mh.GetFacilitiesHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/facility/wrapper/{id:[0-9]+}", mh.GetFacilityWrapperHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/subareatype", mh.CreateSubareaTypeHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/subareatype/{id:[0-9]+}", mh.UpdateSubareaTypeHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/subareatype/{id:[0-9]+}/{version:[0-9]+}", mh.DeleteSubareaTypeHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/subareatype/id/{id:[0-9]+}", mh.GetSubareaTypeHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/subareatypes", mh.GetSubareasTypeHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/itemtype", mh.CreateItemTypeHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/itemtype/{id:[0-9]+}", mh.UpdateItemTypeHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/itemtype/{id:[0-9]+}/{version:[0-9]+}", mh.DeleteItemTypeHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/itemtype/id/{id:[0-9]+}", mh.GetItemTypeHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/itemtypes", mh.GetItemTypesHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/subarea", mh.CreateSubareaHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/subarea/{id:[0-9]+}", mh.UpdateSubareaHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/subarea/{id:[0-9]+}/{version:[0-9]+}", mh.DeleteSubareaHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/subarea/id/{id:[0-9]+}", mh.GetSubareaHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/subareas/{facility:[0-9]+}", mh.GetSubareasHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/product", mh.CreateProductHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/product/{id:[0-9]+}", mh.UpdateProductHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/product/{id:[0-9]+}/{version:[0-9]+}", mh.DeleteProductHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/product/id/{id:[0-9]+}", mh.GetProductHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/products", mh.GetProductsHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/item", mh.CreateItemHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/item/{id:[0-9]+}", mh.UpdateItemHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/item/{id:[0-9]+}/{version:[0-9]+}", mh.DeleteItemHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/item/id/{id:[0-9]+}", mh.GetItemHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/items/product/{id:[0-9]+}", mh.GetItemsByProductHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/items/subarea/{id:[0-9]+}", mh.GetItemsBySubareaHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/items/facility/{id:[0-9]+}", mh.GetItemsByFacilityHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/schema", mh.CreateEntitySchemaHandler).Methods("POST")
	mh.rtr.HandleFunc("/api/schema/{name}", mh.UpdateEntitySchemaHandler).Methods("PUT")
	mh.rtr.HandleFunc("/api/schema/{name}/{version:[0-9]+}", mh.DeleteEntitySchemaHandler).Methods("DELETE")
	mh.rtr.HandleFunc("/api/schema/{name}", mh.EntitySchemaHandler).Methods("GET")
	mh.rtr.HandleFunc("/api/schemas", mh.EntitySchemasHandler).Methods("GET")

	mh.rtr.HandleFunc("/api/server/version", mh.ServerVersionHandler).Methods("GET")
}

func (mh *muxHandler) CreateFacilityHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateFacilityRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateFacility(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) UpdateFacilityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facilityId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.UpdateFacilityRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.FacilityId = facilityId

	ctx := getTokenContext(r)
	resp, err := mh.auth.UpdateFacility(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

func (mh *muxHandler) DeleteFacilityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facilityId, _ := strconv.ParseInt(vars["id"], 10, 64)
	version, _ := strconv.ParseInt(vars["version"], 10, 32)

	req := pb.DeleteFacilityRequest{}
	req.FacilityId = facilityId
	req.Version = int32(version)

	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteFacility(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

func (mh *muxHandler) GetFacilityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facilityId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetFacilityRequest{}
	req.FacilityId = facilityId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetFacility(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetFacilityWrapperHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facilityId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetFacilityWrapperRequest{}
	req.FacilityId = facilityId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetFacilityWrapper(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetFacilitiesHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.GetFacilitiesRequest{}
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetFacilities(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) CreateSubareaTypeHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateSubareaTypeRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateSubareaType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) UpdateSubareaTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.UpdateSubareaTypeRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.SubareaTypeId = int32(typeId)

	ctx := getTokenContext(r)

	resp, err := mh.auth.UpdateSubareaType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) DeleteSubareaTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["id"], 10, 64)
	version, _ := strconv.ParseInt(vars["version"], 10, 32)

	req := pb.DeleteSubareaTypeRequest{}
	req.SubareaTypeId = int32(typeId)
	req.Version = int32(version)

	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteSubareaType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}


func (mh *muxHandler) GetSubareaTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetSubareaTypeRequest{}
	req.SubareaTypeId = int32(typeId)
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetSubareaType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetSubareasTypeHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.GetSubareaTypesRequest{}

	ctx := getTokenContext(r)
	resp, err := mh.auth.GetSubareaTypes(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) CreateItemTypeHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateItemTypeRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateItemType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) UpdateItemTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.UpdateItemTypeRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.ItemTypeId = int32(typeId)

	ctx := getTokenContext(r)

	resp, err := mh.auth.UpdateItemType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) DeleteItemTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["id"], 10, 64)
	version, _ := strconv.ParseInt(vars["version"], 10, 32)

	req := pb.DeleteItemTypeRequest{}
	req.ItemTypeId = int32(typeId)
	req.Version = int32(version)

	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteItemType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}


func (mh *muxHandler) GetItemTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetItemTypeRequest{}
	req.ItemTypeId = int32(typeId)
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetItemType(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetItemTypesHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.GetItemTypesRequest{}
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetItemTypes(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) CreateSubareaHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateSubareaRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateSubarea(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) UpdateSubareaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subareaId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.UpdateSubareaRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.SubareaId = subareaId

	ctx := getTokenContext(r)

	resp, err := mh.auth.UpdateSubarea(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) DeleteSubareaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subareaId, _ := strconv.ParseInt(vars["id"], 10, 64)
	version, _ := strconv.ParseInt(vars["version"], 10, 32)

	req := pb.DeleteSubareaRequest{}
	req.SubareaId = subareaId
	req.Version = int32(version)

	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteSubarea(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetSubareaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subareaId, _ := strconv.ParseInt(vars["id"], 10, 64)

	req := pb.GetSubareaRequest{}
	req.SubareaId = subareaId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetSubarea(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

func (mh *muxHandler) GetSubareasHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facilityId, _ := strconv.ParseInt(vars["facility"], 10, 64)
	req := pb.GetSubareasRequest{}
	req.FacilityId = facilityId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetSubareas(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateProductRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateProduct(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.UpdateProductRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.ProductId = productId
	ctx := getTokenContext(r)
	resp, err := mh.auth.UpdateProduct(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productId, _ := strconv.ParseInt(vars["id"], 10, 64)
	version, _ := strconv.ParseInt(vars["version"], 10, 32)

	req := pb.DeleteProductRequest{}
	req.ProductId = productId
	req.Version = int32(version)
	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteProduct(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetProductRequest{}
	req.ProductId = productId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetProduct(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.GetProductsRequest{}
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetProducts(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) CreateItemHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateInventoryItemRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateInventoryItem(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) UpdateItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.UpdateInventoryItemRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.InventoryItemId = itemId

	ctx := getTokenContext(r)
	resp, err := mh.auth.UpdateInventoryItem(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId, _ := strconv.ParseInt(vars["id"], 10, 64)
	version, _ := strconv.ParseInt(vars["version"], 10, 32)
	req := pb.DeleteInventoryItemRequest{}
	req.InventoryItemId = itemId
	req.Version = int32(version)
	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteInventoryItem(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

func (mh *muxHandler) GetItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetInventoryItemRequest{}
	req.InventoryItemId = itemId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetInventoryItem(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetItemsByProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetInventoryItemsByProductRequest{}
	req.ProductId = productId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetInventoryItemsByProduct(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetItemsBySubareaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subareaId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetInventoryItemsBySubareaRequest{}
	req.SubareaId = subareaId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetInventoryItemsBySubarea(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

func (mh *muxHandler) GetItemsByFacilityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	facilityId, _ := strconv.ParseInt(vars["id"], 10, 64)
	req := pb.GetInventoryItemsByFacilityRequest{}
	req.FacilityId = facilityId
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetInventoryItemsByFacility(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}

// Handle CreateEntitySchema. Expects a POST request and CreateEntitySchemaRequest body.
func (mh *muxHandler) CreateEntitySchemaHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.CreateEntitySchemaRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	ctx := getTokenContext(r)
	resp, err := mh.auth.CreateEntitySchema(ctx, &req)

	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

// Handle UpdateEntitySchema. Expects a PUT request and CreateEntitySchemaRequest body.
func (mh *muxHandler) UpdateEntitySchemaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityName, _ := vars["name"]
	req := pb.UpdateEntitySchemaRequest{}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(501)
		return
	}

	err = json.Unmarshal(buf, &req)
	if err != nil {
		w.WriteHeader(502)
		return
	}

	req.EntityName = entityName

	ctx := getTokenContext(r)
	resp, err := mh.auth.UpdateEntitySchema(ctx, &req)

	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

// Handle DeleteEntitySchema. Expects a DELETE request and nil body.
func (mh *muxHandler) DeleteEntitySchemaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityName, _ := vars["name"]
	version, _ := strconv.ParseInt(vars["version"], 10, 32)
	req := pb.DeleteEntitySchemaRequest{}

	req.EntityName = entityName
	req.Version = int32(version)

	ctx := getTokenContext(r)
	resp, err := mh.auth.DeleteEntitySchema(ctx, &req)

	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}


// Handle GetEntitySchema. Expects a GET request and nil  body.
func (mh *muxHandler) EntitySchemaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityName:= vars["name"]
	req := pb.GetEntitySchemaRequest{}
	req.EntityName = entityName
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetEntitySchema(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}

// Handle GetEntitySchemas. Expects a GET request and nil  body.
func (mh *muxHandler) EntitySchemasHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.GetEntitySchemasRequest{}

	ctx := getTokenContext(r)
	resp, err := mh.auth.GetEntitySchemas(ctx, &req)
	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return

}



// Handle GetServerVersion. Expects a GET request and nil body. Does not require valid JWT.
func (mh *muxHandler) ServerVersionHandler(w http.ResponseWriter, r *http.Request) {
	req := pb.GetServerVersionRequest{}
	req.DummyParam = 1
	ctx := getTokenContext(r)
	resp, err := mh.auth.GetServerVersion(ctx, &req)

	if err == nil {
		writeResponse(resp, err, int(resp.GetErrorCode()), w)
		return
	}

	w.WriteHeader(503)

	return
}


// Helper to write method response as json to ResponseWriter.
func writeResponse(resp interface{}, err error, errCode int, w http.ResponseWriter) {
	if err != nil {
		w.WriteHeader(503)
		return
	}

	if errCode == 0 {
		errCode = 200
	}

	jtext, err2 := json.MarshalIndent(resp, "", "  ")
	if err2 != nil {
		w.WriteHeader(504)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(errCode)
	_, _ = w.Write(jtext)
}

// Gets agrpc context that contains the JWT from the Authorization HTTP header, if available.
func getTokenContext(r *http.Request) context.Context {
	ctx := context.Background()
	var token string
	authHeader := r.Header.Get("Authorization")
	if (len(authHeader) > 7) && (authHeader[0:7] == "Bearer ") {
		token = authHeader[7:]
	}

	md := metadata.Pairs("token", token)

	mctx := metadata.NewIncomingContext(ctx, md)
	return mctx
}