package trip

import "github.com/google/uuid"

func (t Trip) validate() error {
	if t.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Reason: "is required"}
	}
	if t.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Reason: "is required"}
	}
	if err := validatePoint("start_point", t.Start); err != nil {
		return err
	}
	if err := validatePoint("end_point", t.End); err != nil {
		return err
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Reason: "must be non-negative"}
	}
	return nil
}

func validatePoint(field string, p Point) error {
	if p.Latitude < -90 || p.Latitude > 90 {
		return &ValidationError{Field: field + ".latitude", Reason: "must be between -90 and 90"}
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return &ValidationError{Field: field + ".longitude", Reason: "must be between -180 and 180"}
	}
	return nil
}
