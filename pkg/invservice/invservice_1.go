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

// Package invservice provides the GRPC service implementation of MServiceInventory.
package invservice

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
	"google.golang.org/grpc"
)

var NotImplemented = errors.New("not implemented")

type invService struct {
	logger    *log.Logger
	db        *sql.DB
	startSecs int64
}

// Get a new invSerice instance.
func NewInvService() *invService {
	svc := invService{}
	svc.startSecs = time.Now().Unix()
	return &svc
}

// Set the logger for the invService instance.
func (s *invService) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Set the database connection for the invService instance.
func (s *invService) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Get a GRPC api server using the invService instance.
func (s *invService) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceInventoryServer(gServer, s)

	}
	return nil
}

// create new facility
func (s *invService) CreateFacility(ctx context.Context, req *pb.CreateFacilityRequest) (*pb.CreateFacilityResponse, error) {
	s.logger.Printf("CreateFacility called, aid: %d, name: %s\n", req.GetMserviceId(), req.GetFacilityName())
	resp := &pb.CreateFacilityResponse{}

	name := strings.TrimSpace(req.GetFacilityName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "facility_name missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_Facility (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, 
		inbMserviceId, chvFacilityName) VALUES(NOW(), NOW(), NOW(), 0, 1, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), name)

	if err == nil {
		facilityId, err := res.LastInsertId()
		if err != nil {
			s.logger.Printf("LastInsertId err: %v\n", err)
		} else {
			s.logger.Printf("facilityId: %d", facilityId)
		}

		resp.FacilityId = facilityId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, nil
}

// update an existing facility
func (s *invService) UpdateFacility(ctx context.Context, req *pb.UpdateFacilityRequest) (*pb.UpdateFacilityResponse, error) {
	s.logger.Printf("UpdateFacility called, aid: %d, facility_id: %d\n", req.GetMserviceId(), req.GetFacilityId())
	resp := &pb.UpdateFacilityResponse{}
	name := strings.TrimSpace(req.GetFacilityName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "facility_name missing"
		return resp, nil
	}

	sqlstring := `UPDATE tb_Facility SET dtmModified = NOW(), intVersion = intVersion + 1, chvFacilityName = ? 
	WHERE inbFacilityId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(name, req.GetFacilityId(), req.GetMserviceId(), req.GetVersion())
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

// delete an existing facility
func (s *invService) DeleteFacility(ctx context.Context, req *pb.DeleteFacilityRequest) (*pb.DeleteFacilityResponse, error) {
	s.logger.Printf("DeleteFacility called, aid: %d, facility_id: %d\n", req.GetMserviceId(), req.GetFacilityId())
	resp := &pb.DeleteFacilityResponse{}

	sqlstring := `UPDATE tb_Facility SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = intVersion + 1
	WHERE inbFacilityId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetFacilityId(), req.GetMserviceId(), req.GetVersion())
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

// get a facility by id
func (s *invService) GetFacility(ctx context.Context, req *pb.GetFacilityRequest) (*pb.GetFacilityResponse, error) {
	s.logger.Printf("GetFacility called, aid: %d, facility_id: %d\n", req.GetMserviceId(), req.GetFacilityId())
	resp := &pb.GetFacilityResponse{}

	gResp, facility := s.GetFacilityHelper(req.GetMserviceId(), req.GetFacilityId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode == 0 {
		resp.Facility = facility
	}

	return resp, nil
}

// get all facilities by mservice_id
func (s *invService) GetFacilities(ctx context.Context, req *pb.GetFacilitiesRequest) (*pb.GetFacilitiesResponse, error) {
	s.logger.Printf("GetFacilities called, aid: %d\n", req.GetMserviceId())
	resp := &pb.GetFacilitiesResponse{}

	sqlstring := `SELECT inbFacilityId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvFacilityName
	FROM tb_Facility WHERE inbMserviceId = ? AND bitIsDeleted = 0`

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
		var facility pb.Facility

		err := rows.Scan(&facility.FacilityId, &created, &modified,
			&facility.Version, &facility.MserviceId, &facility.FacilityName)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		facility.Created = dml.DateTimeFromString(created)
		facility.Modified = dml.DateTimeFromString(modified)
		resp.Facilities = append(resp.Facilities, &facility)
	}

	return resp, nil

}

// get a facility wrapper by id
func (s *invService) GetFacilityWrapper(ctx context.Context, req *pb.GetFacilityWrapperRequest) (*pb.GetFacilityWrapperResponse, error) {
	s.logger.Printf("GetFacilityWrapper called, aid: %d, facility_id: %d\n", req.GetMserviceId(), req.GetFacilityId())
	resp := &pb.GetFacilityWrapperResponse{}

	gResp, facility := s.GetFacilityHelper(req.GetMserviceId(), req.GetFacilityId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage

	if gResp.ErrorCode != 0 {
		return resp, nil
	}

	facilityWrap := convertFacilityToWrapper(facility)

	wraps := make([]*pb.SubareaWrapper, 0)
	subMap := make(map[int64]*pb.SubareaWrapper)

	gResp, subareas := s.GetSubareasHelper(req.GetMserviceId(), req.GetFacilityId())
	if gResp.ErrorCode != 0 {
		resp.ErrorCode = gResp.ErrorCode
		resp.ErrorMessage = gResp.ErrorMessage
		return resp, nil
	}

	for _, subarea := range subareas {
		wrap := convertSubareaToWrapper(subarea)
		wraps = append(wraps, wrap)
		subMap[wrap.GetSubareaId()] = wrap
	}

	for _, wrap := range wraps {
		if wrap.GetParentSubareaId() == 0 {
			facilityWrap.ChildSubareas = append(facilityWrap.ChildSubareas, wrap)
		} else {
			parent, ok := subMap[wrap.GetParentSubareaId()]
			if ok {
				parent.ChildSubareas = append(parent.ChildSubareas, wrap)
			}
		}
	}

	resp.FacilityWrapper = facilityWrap

	return resp, nil
}

// create new subarea type
func (s *invService) CreateSubareaType(ctx context.Context, req *pb.CreateSubareaTypeRequest) (*pb.CreateSubareaTypeResponse, error) {
	s.logger.Printf("CreateSubareaType called, aid: %d, id: %d, name: %s\n", req.GetMserviceId(), req.GetSubareaTypeId(), req.GetSubareaTypeName())
	resp := &pb.CreateSubareaTypeResponse{}

	name := strings.TrimSpace(req.GetSubareaTypeName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "subarea_type_name missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_SubareaType (inbMserviceId, intSubareaTypeId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, 
		intVersion, chvSubareaTypeName) VALUES(?, ?, NOW(), NOW(), NOW(), 0, 1, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	_, err = stmt.Exec(req.GetMserviceId(), req.GetSubareaTypeId(), name)

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

// update an existing subarea type
func (s *invService) UpdateSubareaType(ctx context.Context, req *pb.UpdateSubareaTypeRequest) (*pb.UpdateSubareaTypeResponse, error) {
	s.logger.Printf("UpdateSubareaType called, aid: %d, id: %d, name: %s\n", req.GetMserviceId(), req.GetSubareaTypeId(), req.GetSubareaTypeName())
	resp := &pb.UpdateSubareaTypeResponse{}
	name := strings.TrimSpace(req.GetSubareaTypeName())
	if name == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "subarea_type_name missing"
		return resp, nil
	}

	sqlstring := `UPDATE tb_SubareaType SET dtmModified = NOW(), intVersion = intVersion + 1, chvSubareaTypeName = ?
	WHERE inbMserviceId = ? AND intSubareaTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(name, req.GetMserviceId(), req.GetSubareaTypeId(), req.GetVersion())
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

// delete an existing subarea type
func (s *invService) DeleteSubareaType(ctx context.Context, req *pb.DeleteSubareaTypeRequest) (*pb.DeleteSubareaTypeResponse, error) {
	s.logger.Printf("DeleteSubareaType called, aid: %d, id: %d\n", req.GetMserviceId(), req.GetSubareaTypeId())
	resp := &pb.DeleteSubareaTypeResponse{}

	sqlstring := `UPDATE tb_SubareaType SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = intVersion + 1
	WHERE inbMserviceId = ? AND intSubareaTypeId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetSubareaTypeId(), req.GetVersion())
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

// get a subarea type by id
func (s *invService) GetSubareaType(ctx context.Context, req *pb.GetSubareaTypeRequest) (*pb.GetSubareaTypeResponse, error) {
	s.logger.Printf("GetSubareaType called, aid: %d, subarea_type_id: %d\n", req.GetMserviceId(), req.GetSubareaTypeId())
	resp := &pb.GetSubareaTypeResponse{}

	sqlstring := `SELECT inbMserviceId, intSubareaTypeId, dtmCreated, dtmModified, intVersion, chvSubareaTypeName
	FROM tb_SubareaType WHERE inbMserviceId = ? AND intSubareaTypeId = ? AND bitIsDeleted = 0`

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
	var subtype pb.SubareaType

	err = stmt.QueryRow(req.GetMserviceId(), req.GetSubareaTypeId()).Scan(&subtype.MserviceId, &subtype.SubareaTypeId, &created,
		&modified, &subtype.Version, &subtype.SubareaTypeName)
	if err == nil {
		subtype.Created = dml.DateTimeFromString(created)
		subtype.Modified = dml.DateTimeFromString(modified)
		resp.SubareaType = &subtype
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

// get  subarea types by mservice_id
func (s *invService) GetSubareaTypes(ctx context.Context, req *pb.GetSubareaTypesRequest) (*pb.GetSubareaTypesResponse, error) {
	s.logger.Printf("GetSubareaTypes called, aid: %d\n", req.GetMserviceId())
	resp := &pb.GetSubareaTypesResponse{}

	sqlstring := `SELECT inbMserviceId, intSubareaTypeId, dtmCreated, dtmModified, intVersion, chvSubareaTypeName
	FROM tb_SubareaType WHERE inbMserviceId = ? AND bitIsDeleted = 0`

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
		var subtype pb.SubareaType

		err := rows.Scan(&subtype.MserviceId, &subtype.SubareaTypeId, &created,
			&modified, &subtype.Version, &subtype.SubareaTypeName)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		subtype.Created = dml.DateTimeFromString(created)
		subtype.Modified = dml.DateTimeFromString(modified)

		resp.SubareaTypes = append(resp.SubareaTypes, &subtype)
	}

	return resp, nil
}
