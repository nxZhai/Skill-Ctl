package web

import "embed"

// Dist contains the Vite production build. The placeholder keeps Go builds
// working before the first npm build.
//
//go:embed dist/*
var Dist embed.FS
