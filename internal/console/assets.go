package console

import "embed"

//go:embed web/*
var assets embed.FS

func Asset(name string) ([]byte, error) {
	return assets.ReadFile("web/" + name)
}
