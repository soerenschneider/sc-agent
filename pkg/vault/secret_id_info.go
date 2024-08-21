package vault

import (
	"encoding/json"
	"errors"
	"time"
)

type SecretIdInfo struct {
	Expiration      time.Time
	CreationTime    time.Time
	LastUpdatedTime time.Time
	Ttl             int64
}

func (s SecretIdInfo) GetPercentage() float64 {
	if s.Expiration.IsZero() {
		return 100.
	}

	lifetime := s.Expiration.Sub(s.CreationTime).Seconds()
	secondsUntilExpiration := time.Until(s.Expiration).Seconds()
	return secondsUntilExpiration * 100 / lifetime
}

func ParseLifetimeIdInfo(data map[string]any) (SecretIdInfo, error) {
	info := SecretIdInfo{}

	// parse expirationTime

	expirationTimeRaw, found := data["expiration_time"]
	if !found {
		return info, errors.New("no field 'expiration_time' in response")
	}

	converted, conversionOk := expirationTimeRaw.(string)
	if conversionOk {
		expiration, err := ParseVaultTimestamp(converted)
		if err == nil {
			info.Expiration = expiration
		}
	}

	// parse creationTime

	creationTimeRaw, found := data["creation_time"]
	if !found {
		return info, errors.New("no field 'creation_time' in response")
	}

	converted, conversionOk = creationTimeRaw.(string)
	if conversionOk {
		creationTime, err := ParseVaultTimestamp(converted)
		if err == nil {
			info.CreationTime = creationTime
		}
	}

	// parse lastUpdatedTime

	lastUpdatedTimeRaw, found := data["last_updated_time"]
	if !found {
		return info, errors.New("no field 'last_updated_time' in response")
	}

	converted, conversionOk = lastUpdatedTimeRaw.(string)
	if conversionOk {
		lastUpdatedTime, err := ParseVaultTimestamp(converted)
		if err == nil {
			info.LastUpdatedTime = lastUpdatedTime
		}
	}

	ttlRaw, found := data["secret_id_ttl"]
	if !found {
		return info, errors.New("no field 'secret_id_ttl' in response")
	}
	ttlNumber, conversionOk := ttlRaw.(json.Number)
	if conversionOk {
		ttl, err := ttlNumber.Int64()
		if err != nil {
			return info, err
		}
		info.Ttl = ttl
	}

	return info, nil
}

func ParseVaultTimestamp(timestamp string) (time.Time, error) {
	layout := "2006-01-02T15:04:05.999999999Z07:00"

	parsedTime, err := time.Parse(layout, timestamp)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}
