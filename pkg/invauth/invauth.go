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
	"log"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/inventory/pkg/mserviceinventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"crypto/rsa"
	"io/ioutil"
)

var NotImplemented = errors.New("not implemented")

type invAuth struct {
	logger          *log.Logger
	db              *sql.DB
	rsaPSSPublicKey *rsa.PublicKey
	invService      pb.MServiceInventoryServer
}

// Get a new invAuth instance.
func NewInvAuth(invService pb.MServiceInventoryServer) *invAuth {
	svc := invAuth{}
	svc.invService = invService
	return &svc
}

// Set a logger for the invAuth instance.
func (s *invAuth) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Set the datbase connection for the invAuth instance.
func (s *invAuth) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Set the RSA public key for the invAuth instance to validate the JWT.
func (s *invAuth) SetPublicKey(publicKeyFile string) error {
	publicKey, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		s.logger.Printf("error reading publicKeyFile: %v\n", err)
		return err
	}

	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		s.logger.Printf("error parsing publicKeyFile: %v\n", err)
		return err
	}

	s.rsaPSSPublicKey = parsedKey
	return nil
}

func (s *invAuth) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceInventoryServer(gServer, s)

	}
	return nil
}

// Get the JWT from the GRPC request context.
func (s *invAuth) GetJwtFromContext(ctx context.Context) (*map[string]interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	tokens := md["token"]

	if (tokens == nil) || (len(tokens) == 0) {
		return nil, fmt.Errorf("cannot get token from context")
	}

	tokenString := tokens[0]

	s.logger.Printf("tokenString: %s\n", tokenString)

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
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid json web token")
	}

	claims := map[string]interface{}(token.Claims.(jwt.MapClaims))

	s.logger.Printf("claims: %v\n", claims)

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
func (s *invAuth) CreateFacility(ctx context.Context, req *pb.CreateFacilityRequest) (*pb.CreateFacilityResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasAdminAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.CreateFacility(ctx, req)
		}
	}
	resp := &pb.CreateFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing facility
func (s *invAuth) UpdateFacility(ctx context.Context, req *pb.UpdateFacilityRequest) (*pb.UpdateFacilityResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasAdminAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.UpdateFacility(ctx, req)
		}
	}
	resp := &pb.UpdateFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing facility
func (s *invAuth) DeleteFacility(ctx context.Context, req *pb.DeleteFacilityRequest) (*pb.DeleteFacilityResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasAdminAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.DeleteFacility(ctx, req)
		}
	}
	resp := &pb.DeleteFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a facility by id
func (s *invAuth) GetFacility(ctx context.Context, req *pb.GetFacilityRequest) (*pb.GetFacilityResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetFacility(ctx, req)
		}
	}
	resp := &pb.GetFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all facilities by mservice_id
func (s *invAuth) GetFacilities(ctx context.Context, req *pb.GetFacilitiesRequest) (*pb.GetFacilitiesResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetFacilities(ctx, req)
		}
	}
	resp := &pb.GetFacilitiesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a facility wrapper by id
func (s *invAuth) GetFacilityWrapper(ctx context.Context, req *pb.GetFacilityWrapperRequest) (*pb.GetFacilityWrapperResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetFacilityWrapper(ctx, req)
		}
	}
	resp := &pb.GetFacilityWrapperResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create new subarea type
func (s *invAuth) CreateSubareaType(ctx context.Context, req *pb.CreateSubareaTypeRequest) (*pb.CreateSubareaTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.CreateSubareaType(ctx, req)
		}
	}
	resp := &pb.CreateSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing subarea type
func (s *invAuth) UpdateSubareaType(ctx context.Context, req *pb.UpdateSubareaTypeRequest) (*pb.UpdateSubareaTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.UpdateSubareaType(ctx, req)
		}
	}
	resp := &pb.UpdateSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing subarea type
func (s *invAuth) DeleteSubareaType(ctx context.Context, req *pb.DeleteSubareaTypeRequest) (*pb.DeleteSubareaTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.DeleteSubareaType(ctx, req)
		}
	}
	resp := &pb.DeleteSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a subarea type by id
func (s *invAuth) GetSubareaType(ctx context.Context, req *pb.GetSubareaTypeRequest) (*pb.GetSubareaTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetSubareaType(ctx, req)
		}
	}
	resp := &pb.GetSubareaTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get  subarea types by mservice_id
func (s *invAuth) GetSubareaTypes(ctx context.Context, req *pb.GetSubareaTypesRequest) (*pb.GetSubareaTypesResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetSubareaTypes(ctx, req)
		}
	}
	resp := &pb.GetSubareaTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create new item type
func (s *invAuth) CreateItemType(ctx context.Context, req *pb.CreateItemTypeRequest) (*pb.CreateItemTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.CreateItemType(ctx, req)
		}
	}
	resp := &pb.CreateItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing item type
func (s *invAuth) UpdateItemType(ctx context.Context, req *pb.UpdateItemTypeRequest) (*pb.UpdateItemTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.UpdateItemType(ctx, req)
		}
	}
	resp := &pb.UpdateItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing item type
func (s *invAuth) DeleteItemType(ctx context.Context, req *pb.DeleteItemTypeRequest) (*pb.DeleteItemTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.DeleteItemType(ctx, req)
		}
	}
	resp := &pb.DeleteItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a item type by id
func (s *invAuth) GetItemType(ctx context.Context, req *pb.GetItemTypeRequest) (*pb.GetItemTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetItemType(ctx, req)
		}
	}
	resp := &pb.GetItemTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get  item types by mservice_id
func (s *invAuth) GetItemTypes(ctx context.Context, req *pb.GetItemTypesRequest) (*pb.GetItemTypesResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetItemTypes(ctx, req)
		}
	}
	resp := &pb.GetItemTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new subarea
func (s *invAuth) CreateSubarea(ctx context.Context, req *pb.CreateSubareaRequest) (*pb.CreateSubareaResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.CreateSubarea(ctx, req)
		}
	}
	resp := &pb.CreateSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing subarea
func (s *invAuth) UpdateSubarea(ctx context.Context, req *pb.UpdateSubareaRequest) (*pb.UpdateSubareaResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.UpdateSubarea(ctx, req)
		}
	}
	resp := &pb.UpdateSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing subarea
func (s *invAuth) DeleteSubarea(ctx context.Context, req *pb.DeleteSubareaRequest) (*pb.DeleteSubareaResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.DeleteSubarea(ctx, req)
		}
	}
	resp := &pb.DeleteSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a subarea by id
func (s *invAuth) GetSubarea(ctx context.Context, req *pb.GetSubareaRequest) (*pb.GetSubareaResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetSubarea(ctx, req)
		}
	}
	resp := &pb.GetSubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all subareas by facility_id
func (s *invAuth) GetSubareas(ctx context.Context, req *pb.GetSubareasRequest) (*pb.GetSubareasResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetSubareas(ctx, req)
		}
	}
	resp := &pb.GetSubareasResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new product
func (s *invAuth) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.CreateProduct(ctx, req)
		}
	}
	resp := &pb.CreateProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing product
func (s *invAuth) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.UpdateProduct(ctx, req)
		}
	}
	resp := &pb.UpdateProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing product
func (s *invAuth) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*pb.DeleteProductResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.DeleteProduct(ctx, req)
		}
	}
	resp := &pb.DeleteProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a product by id
func (s *invAuth) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetProduct(ctx, req)
		}
	}
	resp := &pb.GetProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all products by mservice_id
func (s *invAuth) GetProducts(ctx context.Context, req *pb.GetProductsRequest) (*pb.GetProductsResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetProducts(ctx, req)
		}
	}
	resp := &pb.GetProductsResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new inventory item
func (s *invAuth) CreateInventoryItem(ctx context.Context, req *pb.CreateInventoryItemRequest) (*pb.CreateInventoryItemResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.CreateInventoryItem(ctx, req)
		}
	}
	resp := &pb.CreateInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing inventory item
func (s *invAuth) UpdateInventoryItem(ctx context.Context, req *pb.UpdateInventoryItemRequest) (*pb.UpdateInventoryItemResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.UpdateInventoryItem(ctx, req)
		}
	}
	resp := &pb.UpdateInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing inventory item
func (s *invAuth) DeleteInventoryItem(ctx context.Context, req *pb.DeleteInventoryItemRequest) (*pb.DeleteInventoryItemResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasRWAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.DeleteInventoryItem(ctx, req)
		}
	}
	resp := &pb.DeleteInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get an inventory item by id
func (s *invAuth) GetInventoryItem(ctx context.Context, req *pb.GetInventoryItemRequest) (*pb.GetInventoryItemResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetInventoryItem(ctx, req)
		}
	}
	resp := &pb.GetInventoryItemResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all inventory items for a product id
func (s *invAuth) GetInventoryItemsByProduct(ctx context.Context, req *pb.GetInventoryItemsByProductRequest) (*pb.GetInventoryItemsByProductResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetInventoryItemsByProduct(ctx, req)
		}
	}
	resp := &pb.GetInventoryItemsByProductResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all inventory items in a subarea
func (s *invAuth) GetInventoryItemsBySubarea(ctx context.Context, req *pb.GetInventoryItemsBySubareaRequest) (*pb.GetInventoryItemsBySubareaResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetInventoryItemsBySubarea(ctx, req)
		}
	}
	resp := &pb.GetInventoryItemsBySubareaResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all inventory items in a facility
func (s *invAuth) GetInventoryItemsByFacility(ctx context.Context, req *pb.GetInventoryItemsByFacilityRequest) (*pb.GetInventoryItemsByFacilityResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		if HasReadAccess(claims) {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.invService.GetInventoryItemsByFacility(ctx, req)
		}
	}
	resp := &pb.GetInventoryItemsByFacilityResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get current server version and uptime - health check
func (s *invAuth) GetServerVersion(ctx context.Context, req *pb.GetServerVersionRequest) (*pb.GetServerVersionResponse, error) {
	return s.invService.GetServerVersion(ctx, req)
}
