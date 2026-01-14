//go:build !debug

package main

import "github.com/rs/zerolog"

func applyTagsOverrides(*config, zerolog.Logger) {}
