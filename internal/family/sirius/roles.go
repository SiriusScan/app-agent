package sirius

type RoleDescriptor struct {
	ID          string
	Kind        string
	Description string
}

const (
	RoleConnector     = "sirius.connector"
	RoleScanTemplate  = "sirius.scan.template"
	RoleScanInventory = "sirius.scan.inventory"
	RolePeer          = "sirius.peer"
)

func DefaultRoleMap() []RoleDescriptor {
	return []RoleDescriptor{
		{
			ID:          RoleConnector,
			Kind:        "resident",
			Description: "Owns upstream Sirius communications and family-level sync behavior.",
		},
		{
			ID:          RoleScanTemplate,
			Kind:        "worker",
			Description: "Executes template-driven checks and returns Sirius and normalized results.",
		},
		{
			ID:          RoleScanInventory,
			Kind:        "worker",
			Description: "Collects host inventory when separated from template execution.",
		},
		{
			ID:          RolePeer,
			Kind:        "resident",
			Description: "Hosts Sirius peer-plane behavior when explicitly allowed by policy.",
		},
	}
}
