package auth

import (
	"slices"
	"strings"

	"github.com/H4fizWasabie/pims/internal/config"
)

func IsAdmin(cfg *config.Config, email string) bool { return Contains(cfg.MasterAdmins, email) }

func IsIndentApprover(cfg *config.Config, email string) bool {
	return Contains(cfg.IndentApprovers, email)
}

func IsSpecApprover(cfg *config.Config, email string) bool {
	return Contains(cfg.SpecApprovers, email)
}

// Contains reports whether list has s, ignoring case.
func Contains(list []string, s string) bool {
	return slices.ContainsFunc(list, func(v string) bool { return strings.EqualFold(v, s) })
}
