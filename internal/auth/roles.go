package auth

import "github.com/H4fizWasabie/pims/internal/config"

func IsAdmin(cfg *config.Config, email string) bool {
	return contains(cfg.MasterAdmins, email)
}

func IsIndentApprover(cfg *config.Config, email string) bool {
	return contains(cfg.IndentApprovers, email)
}

func IsSpecApprover(cfg *config.Config, email string) bool {
	return contains(cfg.SpecApprovers, email)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if equalsFold(v, s) {
			return true
		}
	}
	return false
}

func equalsFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
