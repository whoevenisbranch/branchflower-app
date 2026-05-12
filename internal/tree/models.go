package tree

import (
	"log"
	"time"
)

type TreeData struct {
	BaseScores  BaseScores
	UIScores    UIScores
	GeneratedAt time.Time
}

type BaseScores struct {
	History   float64 `json:"history"`
	Fullness  float64 `json:"fullness"`
	Stability float64 `json:"stability"`
	Vitality  float64 `json:"vitality"`
	State     string  `json:"state"`
}

func (base *BaseScores) Display() {
	log.Printf("History Score: %.2f", base.History)
	log.Printf("Fullness Score: %.2f", base.Fullness)
	log.Printf("Stability Score: %.2f", base.Stability)
	log.Printf("Vitality Score: %.2f", base.Vitality)
	log.Printf("State: %s", base.State)
}

type UIScores struct {
	Trunk   trunk  `json:"trunk"`
	Canopy  canopy `json:"canopy"`
	Palette string `json:"palette"`
}

func (base *UIScores) Display() {
	log.Printf("Trunk Height %.2f", base.Trunk.Height)
	log.Printf("Trunk Height %.2f", base.Trunk.Width)

	log.Printf("Canopy Radius X %.2f", base.Canopy.RadiusX)
	log.Printf("Canopy Radius Y %.2f", base.Canopy.RadiusY)
	log.Printf("Canopy Density %.2f", base.Canopy.Density)
	log.Printf("Canopy Smoothness %.2f", base.Canopy.Smoothness)
	log.Printf("Canopy Saturation %.2f", base.Canopy.Saturation)
	log.Printf("Canopy Lightness %.2f", base.Canopy.Lightness)

	log.Printf("Palette %s", base.Palette)
}

type derivedAggregates struct {
	currCanopyActiveDays int
	currCanopyHrs        float64
	prevCanopyActiveDays int
	prevCanopyHrs        float64
	recentHalfCanopyHrs  float64
	olderHalfCanopyHrs   float64
	baselineAvgDailyHrs  float64
	expectedCanopyHrs    float64
}

func (da *derivedAggregates) display() {
	log.Printf("Active Days in Current Canopy Window = %d", da.currCanopyActiveDays)
	log.Printf("Active Days in Previous Canopy Window = %d", da.prevCanopyActiveDays)

	log.Printf("Baseline Average Daily Hours = %.2f", da.baselineAvgDailyHrs)
	log.Printf("Expected Canopy Hours = %.2f", da.expectedCanopyHrs)
	log.Printf("Current Canopy Hours = %.2f [1st: %.2f, 2nd:%.2f]", da.currCanopyHrs, da.recentHalfCanopyHrs, da.olderHalfCanopyHrs)
	log.Printf("Previous Canopy Hours = %.2f", da.prevCanopyHrs)
}

type trunk struct {
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
}

type canopy struct {
	RadiusX    float64 `json:"radiusX"`
	RadiusY    float64 `json:"radiusY"`
	Density    float64 `json:"rensity"`
	Smoothness float64 `json:"smoothness"`
	Saturation float64 `json:"saturation"`
	Lightness  float64 `json:"lightness"`
}