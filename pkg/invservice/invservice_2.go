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

package invservice

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
)

// create new item type
func (s *invService) CreateItemType(ctx context.Context, req *pb.CreateItemTypeRequest) (*pb.CreateItemTypeResponse, error) {
	s.logger.Printf("CreateItemType called, aid: %d, id: %d, name: %s\n", req.GetMserviceId(), req.GetItemTypeId(), req.GetItemTypeName())
	resp := &pb.CreateItemTypeResponse{}

	name := strings.TrimSpace(req.GetItemTypeName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "item_type_name missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_ItemType (inbMserviceId, intItemTypeId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, 
		intVersion, chvItemTypeName) VALUES(?, ?, NOW(), NOW(), NOW(), 0, 1, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	_, err = stmt.Exec(req.GetMserviceId(), req.GetItemTypeId(), name)

	if err == nil {
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update an existing item type
func (s *invService) UpdateItemType(ctx context.Context, req *pb.UpdateItemTypeRequest) (*pb.UpdateItemTypeResponse, error) {
	s.logger.Printf("UpdateItemType called, aid: %d, id: %d, name: %s\n", req.GetMserviceId(), req.GetItemTypeId(), req.GetItemTypeName())
	resp := &pb.UpdateItemTypeResponse{}

	name := strings.TrimSpace(req.GetItemTypeName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "item_type_name missing"
		return resp, nil
	}

	sqlstring := `UPDATE tb_ItemType SET dtmModified = NOW(), intVersion = intVersion + 1, chvItemTypeName = ? WHERE inbMserviceId = ? 
	AND intItemTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(name, req.GetMserviceId(), req.GetItemTypeId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// delete an existing item type
func (s *invService) DeleteItemType(ctx context.Context, req *pb.DeleteItemTypeRequest) (*pb.DeleteItemTypeResponse, error) {
	s.logger.Printf("DeleteItemType called, aid: %d, id: %d\n", req.GetMserviceId(), req.GetItemTypeId())
	resp := &pb.DeleteItemTypeResponse{}

	sqlstring := `UPDATE tb_ItemType SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = intVersion + 1
	WHERE inbMserviceId = ? AND intItemTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetItemTypeId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// get a item type by id
func (s *invService) GetItemType(ctx context.Context, req *pb.GetItemTypeRequest) (*pb.GetItemTypeResponse, error) {
	s.logger.Printf("GetItemType called, aid: %d, item_type_id: %d\n", req.GetMserviceId(), req.GetItemTypeId())
	resp := &pb.GetItemTypeResponse{}

	sqlstring := `SELECT inbMserviceId, intItemTypeId, dtmCreated, dtmModified, intVersion, chvItemTypeName
	FROM tb_ItemType WHERE inbMserviceId = ? AND intItemTypeId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created string
	var modified string
	var itemtype pb.ItemType

	err = stmt.QueryRow(req.GetMserviceId(), req.GetItemTypeId()).Scan(&itemtype.MserviceId, &itemtype.ItemTypeId, &created,
		&modified, &itemtype.Version, &itemtype.ItemTypeName)
	if err == nil {
		itemtype.Created = dml.DateTimeFromString(created)
		itemtype.Modified = dml.DateTimeFromString(modified)
		resp.ItemType = &itemtype
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, nil
}

// get  item types by mservice_id
func (s *invService) GetItemTypes(ctx context.Context, req *pb.GetItemTypesRequest) (*pb.GetItemTypesResponse, error) {
	s.logger.Printf("GetItemTypes called, aid: %d\n", req.GetMserviceId())
	resp := &pb.GetItemTypesResponse{}

	sqlstring := `SELECT inbMserviceId, intItemTypeId, dtmCreated, dtmModified, intVersion, chvItemTypeName
	FROM tb_ItemType WHERE inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())

	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created string
		var modified string
		var itemtype pb.ItemType

		err := rows.Scan(&itemtype.MserviceId, &itemtype.ItemTypeId, &created,
			&modified, &itemtype.Version, &itemtype.ItemTypeName)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		itemtype.Created = dml.DateTimeFromString(created)
		itemtype.Modified = dml.DateTimeFromString(modified)

		resp.ItemTypes = append(resp.ItemTypes, &itemtype)
	}

	return resp, nil
}

// create a new subarea
func (s *invService) CreateSubarea(ctx context.Context, req *pb.CreateSubareaRequest) (*pb.CreateSubareaResponse, error) {
	s.logger.Printf("CreateSubarea called, aid: %d, name: %s\n", req.GetMserviceId(), req.GetSubareaName())
	resp := &pb.CreateSubareaResponse{}

	name := strings.TrimSpace(req.GetSubareaName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "subarea_name missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_Subarea (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, 
		inbFacilityId, inbParentSubareaId, intPosition, intSubareaTypeId, chvSubareaName) 
		VALUES(NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetFacilityId(), req.GetParentSubareaId(), req.GetPosition(),
		req.GetSubareaTypeId(), name)

	if err == nil {
		subareaId, err := res.LastInsertId()
		if err != nil {
			s.logger.Printf("LastInsertId err: %v\n", err)
		} else {
			s.logger.Printf("subareaId: %d", subareaId)
		}

		resp.SubareaId = subareaId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update an existing subarea
func (s *invService) UpdateSubarea(ctx context.Context, req *pb.UpdateSubareaRequest) (*pb.UpdateSubareaResponse, error) {
	s.logger.Printf("UpdateSubarea called, aid: %d, name: %s\n", req.GetMserviceId(), req.GetSubareaName())
	resp := &pb.UpdateSubareaResponse{}

	name := strings.TrimSpace(req.GetSubareaName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "subarea_name missing"
		return resp, nil
	}

	sqlstring := `UPDATE tb_Subarea SET dtmModified = NOW(), intVersion = intVersion + 1, inbParentSubareaId = ?, 
	intPosition = ?, intSubareaTypeId = ?, chvSubareaName = ? WHERE inbSubareaId = ? AND inbMserviceId = ? 
	AND intVersion= ? AND bitIsDeleted= 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetParentSubareaId(), req.GetPosition(), req.GetSubareaTypeId(), req.GetSubareaName(),
		req.GetSubareaId(), req.GetMserviceId(), req.GetVersion())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// delete an existing subarea
func (s *invService) DeleteSubarea(ctx context.Context, req *pb.DeleteSubareaRequest) (*pb.DeleteSubareaResponse, error) {
	s.logger.Printf("DeleteSubarea called, aid: %d, subareaId: %d\n", req.GetMserviceId(), req.GetSubareaId())
	resp := &pb.DeleteSubareaResponse{}

	sqlstring := `UPDATE tb_Subarea SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = intVersion + 1
	WHERE inbSubareaId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetSubareaId(), req.GetMserviceId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// get a subarea by id
func (s *invService) GetSubarea(ctx context.Context, req *pb.GetSubareaRequest) (*pb.GetSubareaResponse, error) {
	s.logger.Printf("GetSubarea called, aid: %d, subareaId: %d\n", req.GetMserviceId(), req.GetSubareaId())
	resp := &pb.GetSubareaResponse{}

	sqlstring := `SELECT s.inbSubareaId, s.dtmCreated, s.dtmModified, s.intVersion, s.inbMserviceId, s.inbFacilityId, 
	s.inbParentSubareaId, s.intPosition, s.intSubareaTypeId, s.chvSubareaName, f.chvFacilityName, t.chvSubareaTypeName 
	FROM tb_Subarea AS s 
	LEFT JOIN tb_Facility AS f ON s.inbFacilityId = f.inbFacilityId
	LEFT JOIN tb_SubareaType AS t ON  s.inbMserviceId = t.inbMserviceId AND s.intSubareaTypeId = t.intSubareaTypeId
	WHERE s.inbMserviceId = ? AND s.inbSubareaId = ? AND s.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created string
	var modified string
	var subarea pb.Subarea
	var facility sql.NullString
	var subtype sql.NullString

	err = stmt.QueryRow(req.GetMserviceId(), req.GetSubareaId()).Scan(&subarea.SubareaId, &created, &modified, &subarea.Version, &subarea.MserviceId,
		&subarea.FacilityId, &subarea.ParentSubareaId, &subarea.Position, &subarea.SubareaTypeId, &subarea.SubareaName, &facility, &subtype)

	if err == nil {
		subarea.Created = dml.DateTimeFromString(created)
		subarea.Modified = dml.DateTimeFromString(modified)
		if facility.Valid {
			subarea.FacilityName = facility.String
		}
		if subtype.Valid {
			subarea.SubareaTypeName = subtype.String
		}
		resp.Subarea = &subarea
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
	}

	return resp, nil
}

// get all subareas by facility_id
func (s *invService) GetSubareas(ctx context.Context, req *pb.GetSubareasRequest) (*pb.GetSubareasResponse, error) {
	s.logger.Printf("GetSubareas called, aid: %d, facilityId: %d\n", req.GetMserviceId(), req.GetFacilityId())
	resp := &pb.GetSubareasResponse{}

	gResp, subareas := s.GetSubareasHelper(req.GetMserviceId(), req.GetFacilityId())

	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	resp.Subareas = subareas

	return resp, nil
}

// create a new product
func (s *invService) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	s.logger.Printf("CreateProduct called, aid: %d, name: %s\n", req.GetMserviceId(), req.GetProductName())
	resp := &pb.CreateProductResponse{}

	name := strings.TrimSpace(req.GetProductName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "product_name missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_Product (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, 
		chvSku, chvProductName, chvComment) VALUES (NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetSku(), name, req.GetComment())
	if err == nil {
		productId, err := res.LastInsertId()
		if err != nil {
			s.logger.Printf("LastInsertId err: %v\n", err)
		} else {
			s.logger.Printf("productId: %d", productId)
		}

		resp.ProductId = productId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update an existing product
func (s *invService) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	s.logger.Printf("UpdateProduct called, aid: %d, productId: %d\n", req.GetMserviceId(), req.GetProductId())
	resp := &pb.UpdateProductResponse{}

	name := strings.TrimSpace(req.GetProductName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "product_name missing"
		return resp, nil
	}

	sqlstring := `UPDATE tb_Product SET dtmModified = NOW(), intVersion = intVersion + 1, chvSku = ?, chvProductName = ?, chvComment = ?
	WHERE inbProductId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetSku(), name, req.GetComment(), req.GetProductId(), req.GetMserviceId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// delete an existing product
func (s *invService) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	s.logger.Printf("DeleteProduct called, aid: %d, productId: %d\n", req.GetMserviceId(), req.GetProductId())
	resp := &pb.DeleteProductResponse{}

	sqlstring := `UPDATE tb_Product SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = intVersion + 1
	WHERE inbProductId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetProductId(), req.GetMserviceId(), req.GetVersion())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// get a product by id
func (s *invService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	s.logger.Printf("GetProduct called, aid: %d, productId: %d\n", req.GetMserviceId(), req.GetProductId())
	resp := &pb.GetProductResponse{}

	sqlstring := `SELECT inbProductId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvSku, chvProductName, chvComment
	FROM tb_Product WHERE inbProductId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created string
	var modified string
	var product pb.Product

	err = stmt.QueryRow(req.GetProductId(), req.GetMserviceId()).Scan(&product.ProductId, &created, &modified, &product.Version,
		&product.MserviceId, &product.Sku, &product.ProductName, &product.Comment)

	if err == nil {
		product.Created = dml.DateTimeFromString(created)
		product.Modified = dml.DateTimeFromString(modified)
		resp.Product = &product
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
	}

	return resp, nil
}

// get all products by mservice_id
func (s *invService) GetProducts(ctx context.Context, req *pb.GetProductsRequest) (*pb.GetProductsResponse, error) {
	s.logger.Printf("GetProducts called, aid: %d\n", req.GetMserviceId())
	resp := &pb.GetProductsResponse{}

	sqlstring := `SELECT inbProductId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvSku, chvProductName, chvComment
	FROM tb_Product WHERE inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())

	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created string
		var modified string
		var product pb.Product

		err := rows.Scan(&product.ProductId, &created, &modified, &product.Version,
			&product.MserviceId, &product.Sku, &product.ProductName, &product.Comment)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		product.Created = dml.DateTimeFromString(created)
		product.Modified = dml.DateTimeFromString(modified)
		resp.Products = append(resp.Products, &product)
	}

	return resp, nil
}

// create a new inventory item
func (s *invService) CreateInventoryItem(ctx context.Context, req *pb.CreateInventoryItemRequest) (*pb.CreateInventoryItemResponse, error) {
	s.logger.Printf("CreateInventoryItem called, aid: %d\n", req.GetMserviceId())
	resp := &pb.CreateInventoryItemResponse{}

	sqlstring := `INSERT INTO tb_InventoryItem (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, inbSubareaId, 
		intItemTypeId, intQuantity, chvSerialNumber, inbProductId) 
		VALUES (NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetSubareaId(), req.GetItemTypeId(), req.GetQuantity(), req.GetSerialNumber(), req.GetProductId())

	if err == nil {
		itemId, err := res.LastInsertId()
		if err != nil {
			s.logger.Printf("LastInsertId err: %v\n", err)
		} else {
			s.logger.Printf("itemId: %d", itemId)
		}

		resp.InventoryItemId = itemId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update an existing inventory item
func (s *invService) UpdateInventoryItem(ctx context.Context, req *pb.UpdateInventoryItemRequest) (*pb.UpdateInventoryItemResponse, error) {
	s.logger.Printf("UpdateInventoryItem called, aid: %d, itemId: %d\n", req.GetMserviceId(), req.GetInventoryItemId())
	resp := &pb.UpdateInventoryItemResponse{}

	sqlstring := `UPDATE tb_InventoryItem SET dtmModified = NOW(), intVersion = intVersion + 1, inbSubareaId = ?, intItemTypeId = ?, 
	intQuantity = ?, chvSerialNumber = ?, inbProductId = ? WHERE inbInventoryItemId= ? AND inbMserviceId = ? AND intVersion = ? 
	AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetSubareaId(), req.GetItemTypeId(), req.GetQuantity(), req.GetSerialNumber(), req.GetProductId(),
		req.GetInventoryItemId(), req.GetMserviceId(), req.GetVersion())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// delete an existing inventory item
func (s *invService) DeleteInventoryItem(ctx context.Context, req *pb.DeleteInventoryItemRequest) (*pb.DeleteInventoryItemResponse, error) {
	s.logger.Printf("DeleteInventoryItem called, aid: %d, itemId: %d\n", req.GetMserviceId(), req.GetInventoryItemId())
	resp := &pb.DeleteInventoryItemResponse{}

	sqlstring := `UPDATE tb_InventoryItem SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = intVersion + 1
	WHERE inbInventoryItemId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetInventoryItemId(), req.GetMserviceId(), req.GetVersion())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// get an inventory item by id
func (s *invService) GetInventoryItem(ctx context.Context, req *pb.GetInventoryItemRequest) (*pb.GetInventoryItemResponse, error) {
	s.logger.Printf("GetInventoryItem called, aid: %d, itemId: %d\n", req.GetMserviceId(), req.GetInventoryItemId())
	resp := &pb.GetInventoryItemResponse{}

	sqlstring := `SELECT i.inbInventoryItemId, i.dtmCreated, i.dtmModified, i.intVersion, i.inbMserviceId,
	i.inbSubareaId, i.intItemTypeId, i.intQuantity, i.chvSerialNumber, i.inbProductId, t.chvItemTypeName, p.chvProductName
	FROM tb_InventoryItem AS i
	LEFT JOIN tb_ItemType as t ON  i.inbMserviceId = t.inbMserviceId AND i.intItemTypeId = t.intItemTypeId
	LEFT JOIN tb_Product as p ON i.inbProductId = p.inbProductId
	WHERE i.inbInventoryItemId = ? AND i.inbMserviceId = ? AND i.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created string
	var modified string
	var item pb.InventoryItem
	var typeName sql.NullString
	var productName sql.NullString

	err = stmt.QueryRow(req.GetInventoryItemId(), req.GetMserviceId()).Scan(&item.InventoryItemId, &created, &modified,
		&item.Version, &item.MserviceId, &item.SubareaId, &item.ItemTypeId, &item.Quantity, &item.SerialNumber,
		&item.InventoryItemId, &typeName, &productName)

	if err == nil {
		item.Created = dml.DateTimeFromString(created)
		item.Modified = dml.DateTimeFromString(modified)
		if typeName.Valid {
			item.ItemTypeName = typeName.String
		}
		if productName.Valid {
			item.ProductName = productName.String
		}
		resp.InventoryItem = &item
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
	}

	return resp, nil
}

// get all inventory items for a product id
func (s *invService) GetInventoryItemsByProduct(ctx context.Context, req *pb.GetInventoryItemsByProductRequest) (*pb.GetInventoryItemsByProductResponse, error) {
	s.logger.Printf("GetInventoryItemsByProduct called, aid: %d, productId: %d\n", req.GetMserviceId(), req.GetProductId())
	resp := &pb.GetInventoryItemsByProductResponse{}

	sqlstring := `SELECT i.inbInventoryItemId, i.dtmCreated, i.dtmModified, i.intVersion, i.inbMserviceId,
	i.inbSubareaId, i.intItemTypeId, i.intQuantity, i.chvSerialNumber, i.inbProductId, t.chvItemTypeName, p.chvProductName
	FROM tb_InventoryItem AS i
	LEFT JOIN tb_ItemType as t ON  i.inbMserviceId = t.inbMserviceId AND i.intItemTypeId = t.intItemTypeId
	LEFT JOIN tb_Product as p ON i.inbProductId = p.inbProductId
	WHERE i.inbProductId = ? AND i.inbMserviceId = ? AND i.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetProductId(), req.GetMserviceId())

	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created string
		var modified string
		var item pb.InventoryItem
		var typeName sql.NullString
		var productName sql.NullString

		err := rows.Scan(&item.InventoryItemId, &created, &modified,
			&item.Version, &item.MserviceId, &item.SubareaId, &item.ItemTypeId, &item.Quantity, &item.SerialNumber,
			&item.InventoryItemId, &typeName, &productName)
		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		item.Created = dml.DateTimeFromString(created)
		item.Modified = dml.DateTimeFromString(modified)
		if typeName.Valid {
			item.ItemTypeName = typeName.String
		}
		if productName.Valid {
			item.ProductName = productName.String
		}

		resp.InventoryItems = append(resp.InventoryItems, &item)
	}

	return resp, nil
}

// get all inventory items in a subarea
func (s *invService) GetInventoryItemsBySubarea(ctx context.Context, req *pb.GetInventoryItemsBySubareaRequest) (*pb.GetInventoryItemsBySubareaResponse, error) {
	s.logger.Printf("GetInventoryItemsBySubarea called, aid: %d, subareaId: %d\n", req.GetMserviceId(), req.GetSubareaId())

	resp := &pb.GetInventoryItemsBySubareaResponse{}

	sqlstring := `SELECT i.inbInventoryItemId, i.dtmCreated, i.dtmModified, i.intVersion, i.inbMserviceId,
	i.inbSubareaId, i.intItemTypeId, i.intQuantity, i.chvSerialNumber, i.inbProductId, t.chvItemTypeName, p.chvProductName
	FROM tb_InventoryItem AS i
	LEFT JOIN tb_ItemType as t ON  i.inbMserviceId = t.inbMserviceId AND i.intItemTypeId = t.intItemTypeId
	LEFT JOIN tb_Product as p ON i.inbProductId = p.inbProductId
	WHERE i.inbSubareaId = ? AND i.inbMserviceId = ? AND i.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetSubareaId(), req.GetMserviceId())

	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created string
		var modified string
		var item pb.InventoryItem
		var typeName sql.NullString
		var productName sql.NullString

		err := rows.Scan(&item.InventoryItemId, &created, &modified,
			&item.Version, &item.MserviceId, &item.SubareaId, &item.ItemTypeId, &item.Quantity, &item.SerialNumber,
			&item.InventoryItemId, &typeName, &productName)
		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		item.Created = dml.DateTimeFromString(created)
		item.Modified = dml.DateTimeFromString(modified)
		if typeName.Valid {
			item.ItemTypeName = typeName.String
		}
		if productName.Valid {
			item.ProductName = productName.String
		}

		resp.InventoryItems = append(resp.InventoryItems, &item)
	}

	return resp, nil
}

// get all inventory items in a facility
func (s *invService) GetInventoryItemsByFacility(ctx context.Context, req *pb.GetInventoryItemsByFacilityRequest) (*pb.GetInventoryItemsByFacilityResponse, error) {
	s.logger.Printf("GetInventoryItemsByFacility called, aid: %d, facilityId: %d\n", req.GetMserviceId(), req.GetFacilityId())
	resp := &pb.GetInventoryItemsByFacilityResponse{}

	sqlstring := `SELECT i.inbInventoryItemId, i.dtmCreated, i.dtmModified, i.intVersion, i.inbMserviceId,
	i.inbSubareaId, i.intItemTypeId, i.intQuantity, i.chvSerialNumber, i.inbProductId, t.chvItemTypeName, p.chvProductName
	FROM tb_InventoryItem AS i
	JOIN tb_Subarea AS a ON i.inbSubareaId = a.inbSubareaId
	LEFT JOIN tb_ItemType as t ON  i.inbMserviceId = t.inbMserviceId AND i.intItemTypeId = t.intItemTypeId
	LEFT JOIN tb_Product as p ON i.inbProductId = p.inbProductId
	WHERE a.inbFacilityId = ? AND i.inbMserviceId = ? AND i.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetFacilityId(), req.GetMserviceId())

	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created string
		var modified string
		var item pb.InventoryItem
		var typeName sql.NullString
		var productName sql.NullString

		err := rows.Scan(&item.InventoryItemId, &created, &modified,
			&item.Version, &item.MserviceId, &item.SubareaId, &item.ItemTypeId, &item.Quantity, &item.SerialNumber,
			&item.InventoryItemId, &typeName, &productName)
		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		item.Created = dml.DateTimeFromString(created)
		item.Modified = dml.DateTimeFromString(modified)
		if typeName.Valid {
			item.ItemTypeName = typeName.String
		}
		if productName.Valid {
			item.ProductName = productName.String
		}

		resp.InventoryItems = append(resp.InventoryItems, &item)
	}

	return resp, nil
}
