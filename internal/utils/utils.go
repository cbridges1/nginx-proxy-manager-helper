package utils

import (
	"fmt"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/models"
)

func EqualStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, item := range a {
		countA[item]++
	}

	for _, item := range b {
		countB[item]++
	}

	for key, count := range countA {
		if countB[key] != count {
			return false
		}
	}

	return true
}

func EqualDomainConfigs(a, b []models.DomainConfig) bool {
	if len(a) != len(b) {
		return false
	}

	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, config := range a {
		key := fmt.Sprintf("%s~%s~%s", config.Domain, config.Address, config.Port)
		countA[key]++
	}

	for _, config := range b {
		key := fmt.Sprintf("%s~%s~%s", config.Domain, config.Address, config.Port)
		countB[key]++
	}

	for key, count := range countA {
		if countB[key] != count {
			return false
		}
	}

	return true
}
