package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMapSingleTon(t *testing.T) {
	InitServerConfigFromFile("./tools/config.sample.yaml")
	provMap := *GetProvincieMapInstance()
	//log.Info().Msgf(provMap)
	//s := provMap.Map["AP"]
	assert.Equal(t, provMap.Map["AP"], Provincia{"Ascoli-Piceno", "Centro"})
	assert.Equal(t, provMap.Map["BL"], Provincia{"Belluno", "Nord-Est"})
	assert.Equal(t, provMap.Map["BN"], Provincia{"Benevento", "Sud"})

}
