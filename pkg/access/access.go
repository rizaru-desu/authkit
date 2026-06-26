// Package access implements Better Auth-style access control: a statement of
// resources→actions, named roles granting subsets, and permission checks.
package access

// Resources and actions for the admin plugin (Better Auth default statement).
const (
	ResourceUser    = "user"
	ResourceSession = "session"

	ActionCreate      = "create"
	ActionList        = "list"
	ActionSetRole     = "set-role"
	ActionBan         = "ban"
	ActionImpersonate = "impersonate"
	ActionDelete      = "delete"
	ActionSetPassword = "set-password"
	ActionRevoke      = "revoke"
)

// Statement maps each resource to the actions that exist for it.
type Statement map[string][]string

// Role maps each resource to the actions a role is granted.
type Role map[string][]string

// Controller holds the statement and named roles and checks permissions.
type Controller struct {
	statement Statement
	roles     map[string]Role
}

// New creates a Controller for the given statement.
func New(statement Statement) *Controller {
	return &Controller{statement: statement, roles: make(map[string]Role)}
}

// SetRole registers (or replaces) a role's grants. Returns c for chaining.
func (c *Controller) SetRole(name string, role Role) *Controller {
	c.roles[name] = role
	return c
}

// HasRole reports whether a role is defined.
func (c *Controller) HasRole(name string) bool {
	_, ok := c.roles[name]
	return ok
}

// Check reports whether roleName is granted ALL requested actions on ALL
// requested resources. An unknown role is denied; an empty request is allowed.
func (c *Controller) Check(roleName string, requested map[string][]string) bool {
	role, ok := c.roles[roleName]
	if !ok {
		return false
	}
	for resource, actions := range requested {
		granted, ok := role[resource]
		if !ok {
			return false
		}
		for _, want := range actions {
			if !contains(granted, want) {
				return false
			}
		}
	}
	return true
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// DefaultController builds the admin-plugin access control with three roles:
//   - admin:     full access
//   - moderator: list/ban users and list/revoke sessions (example custom role)
//   - user:      no admin permissions
//
// Add or change roles here to customise permissions by role.
func DefaultController() *Controller {
	statement := Statement{
		ResourceUser:    {ActionCreate, ActionList, ActionSetRole, ActionBan, ActionImpersonate, ActionDelete, ActionSetPassword},
		ResourceSession: {ActionList, ActionRevoke},
	}

	return New(statement).
		SetRole("admin", Role{
			ResourceUser:    {ActionCreate, ActionList, ActionSetRole, ActionBan, ActionImpersonate, ActionDelete, ActionSetPassword},
			ResourceSession: {ActionList, ActionRevoke},
		}).
		SetRole("moderator", Role{
			ResourceUser:    {ActionList, ActionBan},
			ResourceSession: {ActionList, ActionRevoke},
		}).
		SetRole("user", Role{})
}
