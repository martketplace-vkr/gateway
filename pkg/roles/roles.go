package roles

const (
	Client = 1
	Vendor = 1 << 1
	Admin  = 1 << 2
	All    = Client | Vendor
)

const (
	RoleClient = "client"
	RoleVendor = "vendor"
	RoleAdmin  = "admin"
	RoleAll    = "all"
)

var roles = map[string]struct{}{
	RoleClient: {},
	RoleVendor: {},
	RoleAdmin:  {},
	RoleAll:    {},
}

var RoleMap = map[string]int64{
	RoleClient: Client,
	RoleVendor: Vendor,
	RoleAdmin:  Admin,
}

func Exists(role string) bool {
	_, ok := roles[role]

	return ok
}

func GetAccess(role ...int64) int64 {
	var sum int64
	for _, v := range role {
		sum |= v
	}

	return sum
}

func CheckAccess(role int64, req int64) bool {
	if req == 0 {
		return role == Admin
	}

	access := req & role

	return access == role || access == req
}
