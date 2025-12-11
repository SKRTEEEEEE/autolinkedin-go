package database

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectID utilities

// GenerateObjectID generates a new MongoDB ObjectID
func GenerateObjectID() primitive.ObjectID {
	return primitive.NewObjectID()
}

// GenerateObjectIDString generates a new MongoDB ObjectID as hex string
func GenerateObjectIDString() string {
	return primitive.NewObjectID().Hex()
}

// IsValidObjectID checks if a string is a valid MongoDB ObjectID
func IsValidObjectID(id string) bool {
	if id == "" {
		return false
	}
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}

// ObjectIDFromString converts a hex string to ObjectID
func ObjectIDFromString(id string) (primitive.ObjectID, error) {
	if id == "" {
		return primitive.NilObjectID, fmt.Errorf("empty ObjectID string")
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid ObjectID format: %w", err)
	}

	return objID, nil
}

// ObjectIDsFromStrings converts multiple hex strings to ObjectIDs
func ObjectIDsFromStrings(ids []string) ([]primitive.ObjectID, error) {
	objectIDs := make([]primitive.ObjectID, len(ids))
	for i, id := range ids {
		objID, err := ObjectIDFromString(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ObjectID at index %d: %w", i, err)
		}
		objectIDs[i] = objID
	}
	return objectIDs, nil
}

// Date utilities

// Now returns the current time
func Now() time.Time {
	return time.Now().UTC()
}

// ToMongoDate converts a time.Time to MongoDB date format
func ToMongoDate(t time.Time) primitive.DateTime {
	return primitive.NewDateTimeFromTime(t)
}

// FormatDate formats a time.Time to ISO 8601 string
func FormatDate(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseDate parses an ISO 8601 string to time.Time
func ParseDate(dateStr string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %w", err)
	}
	return t.UTC(), nil
}

// Query builder utilities

// BuildEqualityQuery builds a simple equality query
func BuildEqualityQuery(field string, value interface{}) bson.M {
	return bson.M{field: value}
}

// BuildRangeQuery builds a range query (gte and lte)
func BuildRangeQuery(field string, min, max interface{}) bson.M {
	query := bson.M{}
	conditions := bson.M{}

	if min != nil {
		conditions["$gte"] = min
	}
	if max != nil {
		conditions["$lte"] = max
	}

	if len(conditions) > 0 {
		query[field] = conditions
	}

	return query
}

// BuildInQuery builds an $in query
func BuildInQuery(field string, values []interface{}) bson.M {
	return bson.M{
		field: bson.M{"$in": values},
	}
}

// BuildRegexQuery builds a regex query
func BuildRegexQuery(field, pattern string) bson.M {
	return bson.M{
		field: bson.M{"$regex": pattern},
	}
}

// BuildExistsQuery builds an existence check query
func BuildExistsQuery(field string, exists bool) bson.M {
	return bson.M{
		field: bson.M{"$exists": exists},
	}
}

// BuildOrQuery builds an OR query combining multiple conditions
func BuildOrQuery(conditions ...bson.M) bson.M {
	if len(conditions) == 0 {
		return bson.M{}
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	return bson.M{"$or": conditions}
}

// BuildAndQuery builds an AND query combining multiple conditions
func BuildAndQuery(conditions ...bson.M) bson.M {
	if len(conditions) == 0 {
		return bson.M{}
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	return bson.M{"$and": conditions}
}

// Aggregation pipeline builders

// MatchStage creates a $match aggregation stage
func MatchStage(filter bson.M) bson.M {
	return bson.M{"$match": filter}
}

// GroupStage creates a $group aggregation stage
func GroupStage(id interface{}, fields bson.M) bson.M {
	group := bson.M{"_id": id}
	for k, v := range fields {
		group[k] = v
	}
	return bson.M{"$group": group}
}

// SortStage creates a $sort aggregation stage
func SortStage(sortFields bson.D) bson.M {
	return bson.M{"$sort": sortFields}
}

// LimitStage creates a $limit aggregation stage
func LimitStage(limit int) bson.M {
	return bson.M{"$limit": limit}
}

// SkipStage creates a $skip aggregation stage
func SkipStage(skip int) bson.M {
	return bson.M{"$skip": skip}
}

// LookupStage creates a $lookup aggregation stage (join)
func LookupStage(from, localField, foreignField, as string) bson.M {
	return bson.M{
		"$lookup": bson.M{
			"from":         from,
			"localField":   localField,
			"foreignField": foreignField,
			"as":           as,
		},
	}
}

// ProjectStage creates a $project aggregation stage
func ProjectStage(fields bson.M) bson.M {
	return bson.M{"$project": fields}
}

// UnwindStage creates an $unwind aggregation stage
func UnwindStage(path string) bson.M {
	return bson.M{"$unwind": path}
}

// Error wrapping utilities

// WrapError wraps an error with context
func WrapError(context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsNotFoundError checks if an error is a "not found" error
func IsNotFoundError(err error) bool {
	return err == ErrEntityNotFound
}

// IsDuplicateError checks if an error is a duplicate key error
func IsDuplicateError(err error) bool {
	return err == ErrEntityAlreadyExists
}

// Pagination utilities

// CalculatePagination calculates skip and limit for pagination
func CalculatePagination(page, pageSize int) (skip, limit int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	skip = int64((page - 1) * pageSize)
	limit = int64(pageSize)
	return
}

// CalculateTotalPages calculates total pages from total items and page size
func CalculateTotalPages(totalItems int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		pages++
	}
	return pages
}

// PaginationInfo holds pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginationInfo creates pagination metadata
func NewPaginationInfo(page, pageSize int, totalItems int64) *PaginationInfo {
	totalPages := CalculateTotalPages(totalItems, pageSize)

	return &PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// BSON conversion utilities

// ToBSON converts a struct or map to BSON
func ToBSON(v interface{}) (bson.M, error) {
	if v == nil {
		return nil, fmt.Errorf("cannot convert nil to BSON")
	}

	data, err := bson.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to BSON: %w", err)
	}

	var result bson.M
	err = bson.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	return result, nil
}

// FromBSON converts BSON to a struct
func FromBSON(data bson.M, v interface{}) error {
	if v == nil {
		return fmt.Errorf("target cannot be nil")
	}

	bsonData, err := bson.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal BSON: %w", err)
	}

	err = bson.Unmarshal(bsonData, v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to struct: %w", err)
	}

	return nil
}

// Update builder utilities

// SetUpdate creates a $set update operation
func SetUpdate(updates bson.M) bson.M {
	return bson.M{"$set": updates}
}

// UnsetUpdate creates an $unset update operation
func UnsetUpdate(fields ...string) bson.M {
	unset := bson.M{}
	for _, field := range fields {
		unset[field] = ""
	}
	return bson.M{"$unset": unset}
}

// IncUpdate creates an $inc update operation
func IncUpdate(field string, value interface{}) bson.M {
	return bson.M{"$inc": bson.M{field: value}}
}

// PushUpdate creates a $push update operation
func PushUpdate(field string, value interface{}) bson.M {
	return bson.M{"$push": bson.M{field: value}}
}

// PullUpdate creates a $pull update operation
func PullUpdate(field string, value interface{}) bson.M {
	return bson.M{"$pull": bson.M{field: value}}
}

// AddToSetUpdate creates an $addToSet update operation
func AddToSetUpdate(field string, value interface{}) bson.M {
	return bson.M{"$addToSet": bson.M{field: value}}
}
