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

// Package invauth provides authorization for each GRPC method in MServiceInventory.
// The JWT extracted from the GRPC request context is used for each delegating method.
package invauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"crypto/rsa"
	"io/ioutil"
)

var NotImplemented = errors.New("not implemented")

const (
	tokenExpiredMatch = "Token is expired"
	tokenExpiredMessage = "token is expired"
)

type InvAuth struct {
	logger          log.Logger
	db              *sql.DB
	rsaPSSPublicKey *rsa.PublicKey
	invService      pb.MServiceInventoryServer
}

// Get a new invAuth instance.
func NewInvAuth(invService pb.MServiceInventoryServer) *InvAuth {
	svc := InvAuth{}
	svc.invService = invService
	return &svc
}

// Set a logger for the invAuth instance.
func (s *InvAuth) SetLogger(logger log.Logger) {
	s.logger = logger
}

// Set the database connection for the invAuth instance.
func (s *InvAuth) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Set the RSA public key for the invAuth instance to validate the JWT.
func (s *InvAuth) SetPublicKey(publicKeyFile string) error {
	publicKey, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		level.Error(s.logger).Log("what", "reading publicKeyFile", "error", err)
		return err
	}

	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		level.Error(s.logger).Log("what", "ParseRSAPublicKeyFromPEM", "error", err)
		return err
	}

	s.rsaPSSPublicKey = parsedKey
	return nil
}

func (s *InvAuth) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceInventoryServer(gServer, s)

	}
	return nil
}

// Get the JWT from the GRPC request context.
func (s *InvAuth) GetJwtFromContext(ctx context.Context) (*map[string]interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	tokens := md["token"]

	if (tokens == nil) || (len(tokens) == 0) {
		return nil, fmt.Errorf("cannot get token from context")
	}

	tokenString := tokens[0]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		method := token.Method.Alg()
		if method != "PS256" {

			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// return []byte(mySigningKey), nil
		return s.rsaPSSPublicKey, nil
	})

	if err != nil {
		level.Debug(s.logger).Log("jwt_error", err)
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("expired json web token")
	}

	claims := map[string]interface{}(token.Claims.(jwt.MapClaims))

	return &claims, nil

}

// Helper to see if JWT provides admin access.
func HasAdminAccess(claims *map[string]interface{}) bool {
	invsvc := GetStringFromClaims(claims, "invsvc")
	if invsvc == "invadmin" {
		return true
	}

	return false
}

// Helper to see if JWT provides read/write access.
func HasRWAccess(claims *map[string]interface{}) bool {
	invsvc := GetStringFromClaims(claims, "invsvc")
	if (invsvc == "invadmin") || (invsvc == "invrw") {
		return true
	}

	return false
}

// Helper to see if JWT provides read access.
func HasReadAccess(claims *map[string]interface{}) bool {
	invsvc := GetStringFromClaims(claims, "invsvc")
	if (invsvc == "invadmin") || (invsvc == "invrw") || (invsvc == "invro") {
		return true
	}

	return false
}

// Helper to get a claim value as int64.
func GetInt64FromClaims(claims *map[string]interface{}, key string) int64 {
	var val int64

	if claims != nil {
		cval := (*claims)[key]
		if fval, ok := cval.(float64); ok {
			val = int64(fval)
		}
	}

	return val
}

// Helper to get claim value as string.
func GetStringFromClaims(claims *map[string]interface{}, key string) string {
	var val string

	if claims != nil {
		cval := (*claims)[key]
		if sval, ok := cval.(string); ok {
			val = sval
		}
	}

	return val
}

// create new facility
func (s *InvAuth) CreateFacility(ctx context.Context, req *pb.CreateFacilityRequest) (*pb.CreateFacilityResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasAdminAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.CreateFacility(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateFacility",
		"facility", req.GetFacilityName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing facility
func (s *InvAuth) UpdateFacility(ctx context.Context, req *pb.UpdateFacilityRequest) (*pb.UpdateFacilityResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasAdminAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.UpdateFacility(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateFacility",
		"facility", req.GetFacilityName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing facility
func (s *InvAuth) DeleteFacility(ctx context.Context, req *pb.DeleteFacilityRequest) (*pb.DeleteFacilityResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasAdminAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.DeleteFacility(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteFacility",
		"facilityid", req.GetFacilityId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a facility by id
func (s *InvAuth) GetFacility(ctx context.Context, req *pb.GetFacilityRequest) (*pb.GetFacilityResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetFacility(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetFacility",
		"facilityid", req.GetFacilityId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all facilities by mservice_id
func (s *InvAuth) GetFacilities(ctx context.Context, req *pb.GetFacilitiesRequest) (*pb.GetFacilitiesResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetFacilitiesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetFacilities(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetFacilities",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a facility wrapper by id
func (s *InvAuth) GetFacilityWrapper(ctx context.Context, req *pb.GetFacilityWrapperRequest) (*pb.GetFacilityWrapperResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetFacilityWrapperResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.GetFacilityWrapper(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
		resp.ErrorCode = 498
		resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetFacilityWrapper",
		"facilityid", req.GetFacilityId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create new subarea type
func (s *InvAuth) CreateSubareaType(ctx context.Context, req *pb.CreateSubareaTypeRequest) (*pb.CreateSubareaTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.CreateSubareaType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateSubareaType",
		"subareatype", req.GetSubareaTypeName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing subarea type
func (s *InvAuth) UpdateSubareaType(ctx context.Context, req *pb.UpdateSubareaTypeRequest) (*pb.UpdateSubareaTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.UpdateSubareaType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateSubareaType",
		"subareatype", req.GetSubareaTypeName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing subarea type
func (s *InvAuth) DeleteSubareaType(ctx context.Context, req *pb.DeleteSubareaTypeRequest) (*pb.DeleteSubareaTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.DeleteSubareaType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteSubareaType",
		"subareatypeid", req.GetSubareaTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a subarea type by id
func (s *InvAuth) GetSubareaType(ctx context.Context, req *pb.GetSubareaTypeRequest) (*pb.GetSubareaTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetSubareaType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetSubareaType",
		"subareatypeid", req.GetSubareaTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get  subarea types by mservice_id
func (s *InvAuth) GetSubareaTypes(ctx context.Context, req *pb.GetSubareaTypesRequest) (*pb.GetSubareaTypesResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetSubareaTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetSubareaTypes(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetSubareaTypes",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create new item type
func (s *InvAuth) CreateItemType(ctx context.Context, req *pb.CreateItemTypeRequest) (*pb.CreateItemTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.CreateItemType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateItemType",
		"itemtype", req.GetItemTypeName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing item type
func (s *InvAuth) UpdateItemType(ctx context.Context, req *pb.UpdateItemTypeRequest) (*pb.UpdateItemTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.UpdateItemType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateItemType",
		"itemtype", req.GetItemTypeName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing item type
func (s *InvAuth) DeleteItemType(ctx context.Context, req *pb.DeleteItemTypeRequest) (*pb.DeleteItemTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.DeleteItemType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteItemType",
		"itemtypeid", req.GetItemTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a item type by id
func (s *InvAuth) GetItemType(ctx context.Context, req *pb.GetItemTypeRequest) (*pb.GetItemTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetItemType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetItemType",
		"itemtypeid", req.GetItemTypeId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get  item types by mservice_id
func (s *InvAuth) GetItemTypes(ctx context.Context, req *pb.GetItemTypesRequest) (*pb.GetItemTypesResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetItemTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetItemTypes(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetItemTypes",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new subarea
func (s *InvAuth) CreateSubarea(ctx context.Context, req *pb.CreateSubareaRequest) (*pb.CreateSubareaResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.CreateSubarea(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateSubarea",
		"subarea", req.GetSubareaName(),
		"errcode", resp.GetErrorCode(), "duration", duration)


	return resp, err
}

// update an existing subarea
func (s *InvAuth) UpdateSubarea(ctx context.Context, req *pb.UpdateSubareaRequest) (*pb.UpdateSubareaResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.UpdateSubarea(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateSubarea",
		"subarea", req.GetSubareaName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing subarea
func (s *InvAuth) DeleteSubarea(ctx context.Context, req *pb.DeleteSubareaRequest) (*pb.DeleteSubareaResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.DeleteSubarea(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteSubarea",
		"subareaid", req.GetSubareaId(),
		"errcode", resp.GetErrorCode(), "duration", duration)


	return resp, err
}

// get a subarea by id
func (s *InvAuth) GetSubarea(ctx context.Context, req *pb.GetSubareaRequest) (*pb.GetSubareaResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetSubarea(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetSubarea",
		"subareaid", req.GetSubareaId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all subareas by facility_id
func (s *InvAuth) GetSubareas(ctx context.Context, req *pb.GetSubareasRequest) (*pb.GetSubareasResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetSubareasResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetSubareas(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetSubareas",
		"facilityid", req.GetFacilityId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new product
func (s *InvAuth) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.CreateProduct(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateProduct",
		"product", req.GetProductName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing product
func (s *InvAuth) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.UpdateProduct(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateProduct",
		"product", req.GetProductName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing product
func (s *InvAuth) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.DeleteProduct(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteProduct",
		"productid", req.GetProductId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a product by id
func (s *InvAuth) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetProduct(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProduct",
		"productid", req.GetProductId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all products by mservice_id
func (s *InvAuth) GetProducts(ctx context.Context, req *pb.GetProductsRequest) (*pb.GetProductsResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProductsResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.GetProducts(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProducts",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new inventory item
func (s *InvAuth) CreateInventoryItem(ctx context.Context, req *pb.CreateInventoryItemRequest) (*pb.CreateInventoryItemResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.invService.CreateInventoryItem(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateInventoryItem",
		"productid", req.GetProductId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing inventory item
func (s *InvAuth) UpdateInventoryItem(ctx context.Context, req *pb.UpdateInventoryItemRequest) (*pb.UpdateInventoryItemResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.UpdateInventoryItem(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateInventoryItem",
		"itemid", req.GetInventoryItemId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing inventory item
func (s *InvAuth) DeleteInventoryItem(ctx context.Context, req *pb.DeleteInventoryItemRequest) (*pb.DeleteInventoryItemResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.DeleteInventoryItem(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteInventoryItem",
		"itemid", req.GetInventoryItemId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get an inventory item by id
func (s *InvAuth) GetInventoryItem(ctx context.Context, req *pb.GetInventoryItemRequest) (*pb.GetInventoryItemResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.GetInventoryItem(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetInventoryItem",
		"itemid", req.GetInventoryItemId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all inventory items for a product id
func (s *InvAuth) GetInventoryItemsByProduct(ctx context.Context, req *pb.GetInventoryItemsByProductRequest) (*pb.GetInventoryItemsByProductResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetInventoryItemsByProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.GetInventoryItemsByProduct(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetInventoryItemsByProduct",
		"productid", req.GetProductId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all inventory items in a subarea
func (s *InvAuth) GetInventoryItemsBySubarea(ctx context.Context, req *pb.GetInventoryItemsBySubareaRequest) (*pb.GetInventoryItemsBySubareaResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetInventoryItemsBySubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err =  s.invService.GetInventoryItemsBySubarea(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetInventoryItemsBySubarea",
		"subareaid", req.GetSubareaId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all inventory items in a facility
func (s *InvAuth) GetInventoryItemsByFacility(ctx context.Context, req *pb.GetInventoryItemsByFacilityRequest) (*pb.GetInventoryItemsByFacilityResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetInventoryItemsByFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetInventoryItemsByFacility(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetInventoryItemsByFacility",
		"facilityid", req.GetFacilityId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get current server version and uptime - health check
func (s *InvAuth) GetServerVersion(ctx context.Context, req *pb.GetServerVersionRequest) (*pb.GetServerVersionResponse, error) {
	return s.invService.GetServerVersion(ctx, req)
}
