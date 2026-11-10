package create

import (
	"bytes"
	"encoding/json"

	"github.com/midavuha1/go-course-lab/api"
	"github.com/midavuha1/go-course-lab/internal/domain/trip"
)

const fieldBody = "body"

var (
	requiredFields = []string{"user_id", "driver_id", "start_point", "end_point", "price"}
	pointFields    = []string{"start_point", "end_point"}
	coordFields    = []string{"latitude", "longitude"}
)

func decode(raw []byte) (api.TripData, error) {
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return api.TripData{}, &trip.ValidationError{Field: fieldBody, Reason: "must be a JSON object: " + err.Error()}
	}
	if err := checkRequired(fields); err != nil {
		return api.TripData{}, err
	}

	var body api.TripData
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return api.TripData{}, &trip.ValidationError{Field: fieldBody, Reason: "is invalid: " + err.Error()}
	}
	return body, nil
}

func checkRequired(fields map[string]any) error {
	for _, k := range requiredFields {
		if v, ok := fields[k]; !ok || v == nil {
			return &trip.ValidationError{Field: k, Reason: "is required"}
		}
	}
	for _, k := range pointFields {
		p, ok := fields[k].(map[string]any)
		if !ok {
			return &trip.ValidationError{Field: k, Reason: "must be an object"}
		}
		for _, f := range coordFields {
			if v, ok := p[f]; !ok || v == nil {
				return &trip.ValidationError{Field: k + "." + f, Reason: "is required"}
			}
		}
	}
	return nil
}
