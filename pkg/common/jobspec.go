package common

import (
	"encoding/base64"
	"encoding/json"
)

// EncodeJobSpec encodes a JobSpec into a base64-encoded string.
func EncodeJobSpec(jobSpec JobSpec) (string, error) {
	data, err := json.Marshal(jobSpec)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// DecodeJobSpec decodes a base64-encoded string into a JobSpec.
func DecodeJobSpec(encoded string) (JobSpec, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return JobSpec{}, err
	}
	var jobSpec JobSpec
	err = json.Unmarshal(data, &jobSpec)
	if err != nil {
		return JobSpec{}, err
	}
	return jobSpec, nil
}
