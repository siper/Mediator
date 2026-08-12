package domain

type Setting struct {
	Key   string
	Value string
}

const SettingAutoApproveRequests = "auto_approve_requests"
const SettingRequestsEnabled = "requests_enabled"
const SettingJWTSecret = "jwt_secret"

var PublicSettingKeys = []string{SettingRequestsEnabled}

type SettingRepository interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	List() ([]Setting, error)
}
