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
	"database/sql"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
)

// reusable response struct for specific method response generation.
type genericResponse struct {
	ErrorCode    int32
	ErrorMessage string
}

// Helper to get the facility given the mservice account id and facility id.
func (s *invService) GetFacilityHelper(mserviceId int64, facilityId int64) (*genericResponse, *pb.Facility) {
	resp := &genericResponse{}

	sqlstring := `SELECT inbFacilityId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvFacilityName
	FROM tb_Facility WHERE inbFacilityId= ? AND inbMserviceId = ? AND bitIsDeleted = 0`

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
	var facility pb.Facility

	err = stmt.QueryRow(facilityId, mserviceId).Scan(&facility.FacilityId, &created, &modified,
		&facility.Version, &facility.MserviceId, &facility.FacilityName)

	if err == nil {
		facility.Created = dml.DateTimeFromString(created)
		facility.Modified = dml.DateTimeFromString(modified)
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		s.logger.Printf("queryRow failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, &facility
}

// Helper to get the subareas for a facility.
func (s *invService) GetSubareasHelper(mserviceId int64, facilityId int64) (*genericResponse, []*pb.Subarea) {
	resp := &genericResponse{}
	subareas := make([]*pb.Subarea, 0)

	sqlstring := `SELECT s.inbSubareaId, s.dtmCreated, s.dtmModified, s.intVersion, s.inbMserviceId, s.inbFacilityId, 
	s.inbParentSubareaId, s.intPosition, s.intSubareaTypeId, s.chvSubareaName, f.chvFacilityName, t.chvSubareaTypeName 
	FROM tb_Subarea AS s 
	LEFT JOIN tb_Facility AS f ON s.inbFacilityId = f.inbFacilityId
	LEFT JOIN tb_SubareaType AS t ON s.inbMserviceId = t.inbMserviceId AND s.intSubareaTypeId = t.intSubareaTypeId
	WHERE s.inbMserviceId = ? AND s.inbFacilityId = ? AND s.bitIsDeleted = 0
	ORDER BY s.inbParentSubareaId, s.intPosition`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(mserviceId, facilityId)

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
		var subarea pb.Subarea
		var facility sql.NullString
		var subtype sql.NullString

		err := rows.Scan(&subarea.SubareaId, &created, &modified, &subarea.Version, &subarea.MserviceId,
			&subarea.FacilityId, &subarea.ParentSubareaId, &subarea.Position, &subarea.SubareaTypeId,
			&subarea.SubareaName, &facility, &subtype)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		subarea.Created = dml.DateTimeFromString(created)
		subarea.Modified = dml.DateTimeFromString(modified)
		if facility.Valid {
			subarea.FacilityName = facility.String
		}
		if subtype.Valid {
			subarea.SubareaTypeName = subtype.String
		}

		subareas = append(subareas, &subarea)
	}

	return resp, subareas
}

// Helper to convert Facility to FacilityWrapper.
func convertFacilityToWrapper(facility *pb.Facility) *pb.FacilityWrapper {
	wrap := pb.FacilityWrapper{}
	wrap.FacilityId = facility.GetFacilityId()
	wrap.Created = facility.GetCreated()
	wrap.Modified = facility.GetModified()
	wrap.Deleted = facility.GetDeleted()
	wrap.IsDeleted = facility.GetIsDeleted()
	wrap.Version = facility.GetVersion()
	wrap.MserviceId = facility.GetMserviceId()
	wrap.FacilityName = facility.GetFacilityName()

	return &wrap
}

// Helper top convert Subarea to SubareaWrapper.
func convertSubareaToWrapper(subarea *pb.Subarea) *pb.SubareaWrapper {
	wrap := pb.SubareaWrapper{}
	wrap.SubareaId = subarea.GetSubareaId()
	wrap.Created = subarea.GetCreated()
	wrap.Modified = subarea.GetModified()
	wrap.Deleted = subarea.GetDeleted()
	wrap.IsDeleted = subarea.GetIsDeleted()
	wrap.Version = subarea.GetVersion()
	wrap.MserviceId = subarea.GetMserviceId()
	wrap.FacilityId = subarea.GetFacilityId()
	wrap.FacilityName = subarea.GetFacilityName()
	wrap.ParentSubareaId = subarea.GetParentSubareaId()
	wrap.Position = subarea.GetPosition()
	wrap.SubareaTypeId = subarea.GetSubareaTypeId()
	wrap.SubareaTypeName = subarea.GetSubareaTypeName()
	wrap.SubareaName = subarea.GetSubareaName()

	return &wrap
}
