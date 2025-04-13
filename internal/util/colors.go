package util

import "github.com/fatih/color"

var colorsOptions = map[string]color.Attribute{
	"red":       color.FgHiRed,
	"green":     color.FgGreen,
	"underline": color.Underline,
	"bold":      color.Bold,
	"bgRed":     color.BgRed,
	"bgGreen":   color.BgGreen,
}

func ColorOutput(text string, colorOptions ...string) string {
	attributes := []color.Attribute{}
	for _, option := range colorOptions {
		if o, ok := colorsOptions[option]; ok {
			attributes = append(attributes, o)
		}
	}
	c := color.New(attributes...)
	return c.Sprint(text)
}
