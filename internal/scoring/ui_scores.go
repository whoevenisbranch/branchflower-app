package scoring

import "log"

func DeriveUIScores(baseScores BaseScores) UIScores {

	var uiScores UIScores

	trunk := getUITrunkValues()
	canopy := getUICanopyValues()

	uiScores.Trunk = trunk
	uiScores.Canopy = canopy
	uiScores.Palette = ""

	return uiScores
}

func getUITrunkValues() trunk {
	panic("unimplemented")
}

func getUICanopyValues() canopy {
	panic("unimplemented")
}

type UIScores struct {
	Trunk   trunk  `json:"trunk"`
	Canopy  canopy `json:"canopy"`
	Palette string `json:"palette"`
}

func (base *UIScores) Display() {
	log.Printf("Trunk Height %.2f", base.Trunk.Height)
	log.Printf("Trunk Height %.2f", base.Trunk.Width)

	log.Printf("Canopy %.2f", base.Canopy.RadiusX)
	log.Printf("Canopy %.2f", base.Canopy.RadiusY)
	log.Printf("Canopy %.2f", base.Canopy.Density)
	log.Printf("Canopy %.2f", base.Canopy.Smoothness)
	log.Printf("Canopy %.2f", base.Canopy.Saturation)
	log.Printf("Canopy %.2f", base.Canopy.Lightness)

	log.Printf("Palette %s", base.Palette)
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
