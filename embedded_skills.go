package main

import (
	"embed"
	"io/fs"

	"github.com/JimmyMcBride/brain/internal/skills"
)

//go:embed skills/brain/**
var embeddedSkills embed.FS

func init() {
	bundle, err := fs.Sub(embeddedSkills, "skills/brain")
	if err == nil {
		skills.RegisterBundle(bundle)
	}
}
