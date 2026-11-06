package server

import "strings"

func normalize(u string) string { return strings.ToLower(strings.TrimSpace(u)) }
