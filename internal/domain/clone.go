package domain

import "encoding/json"

func CloneBatch(in *DigitizationBatch) (*DigitizationBatch, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out DigitizationBatch
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
