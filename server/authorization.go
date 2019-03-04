package main

import (
	"sort"

	"github.com/ccontavalli/goutils/misc"
)

type GroupID string

type Group struct {
	Id      GroupID
	APIKeys []APIKey
	Members []string
}

func (group *Group) Contains(credentials *Credentials) bool {
	// If there is an api key check if it matches
	for _, apikey := range group.APIKeys {
		if credentials.APIKey == apikey {
			return true
		}
	}

	// No user? Nothing else left to do...
	if credentials.User == nil {
		return false
	}

	// Be defensive. Since we use special characters like ".", "=", "!"
	// in ACLs, prevent emails <= 3 characters to prevent witty users.
	email := credentials.User.GetEmail()
	if len(email) <= 3 {
		return false
	}

	// Check if the exact user is allowed to view the page.
	if misc.SortedHasString(group.Members, email) {
		return true
	}

	// Same deal as before. "@it" is 3 characters, and it's already
	// way too short, unless you work for the Vatican.
	domain := credentials.User.GetDomain()
	if len(domain) <= 3 {
		return false
	}

	// Check if the domain of the user is allowed to view the page.
	if misc.SortedHasString(group.Members, domain) {
		return true
	}

	return false
}

const kWildcard = "."

func (group *Group) HasWildcard() bool {
	if misc.SortedHasString(group.Members, kWildcard) {
		return true
	}
	return false
}

type Resource interface {
	GetReaders() []GroupID
	GetWriters() []GroupID
}

type AuthorizationManager struct {
	groups []Group
}

func NewAuthorizationManager(groups []Group) *AuthorizationManager {
	//log.Printf("Authorization groups:\n%v", groups)
	for _, group := range groups {
		sort.Strings(group.Members)
	}
	return &AuthorizationManager{groups: groups}
}

func (manager *AuthorizationManager) resolveGroupIDs(credentials *Credentials) []GroupID {
	group_ids := []GroupID{}
	if manager.groups == nil {
		return group_ids
	}
	for _, group := range manager.groups {
		if group.Contains(credentials) {
			group_ids = append(group_ids, group.Id)
		}
	}
	return group_ids
}

func (manager *AuthorizationManager) getWildcardGroups() []GroupID {
	group_ids := []GroupID{}
	if manager.groups == nil {
		return group_ids
	}
	for _, group := range manager.groups {
		if group.HasWildcard() {
			group_ids = append(group_ids, group.Id)
		}
	}
	return group_ids
}

func groupsOverlap(a_groups []GroupID, b_groups []GroupID) bool {
	for _, a := range a_groups {
		for _, b := range b_groups {
			if a == b {
				return true
			}
		}
	}
	return false
}
func (manager *AuthorizationManager) belongsToGroup(credentials *Credentials, group GroupID) bool {
	for _, a := range manager.resolveGroupIDs(credentials) {
		if a == group {
			return true
		}

	}
	return false

}
func (manager *AuthorizationManager) IsAdmin(credentials *Credentials) bool {
	return manager.belongsToGroup(credentials, "team_anpr")
}
func (manager *AuthorizationManager) IsAdminReader(credentials *Credentials) bool {
	return manager.belongsToGroup(credentials, "team_anpr_readers")
}

func (manager *AuthorizationManager) HasReadAccess(resource Resource, credentials *Credentials) bool {
	requestor_gids := manager.resolveGroupIDs(credentials)
	resource_gids := resource.GetReaders()
	return groupsOverlap(requestor_gids, resource_gids)
}

func (manager *AuthorizationManager) HasWriteAccess(resource Resource, credentials *Credentials) bool {
	requestor_gids := manager.resolveGroupIDs(credentials)
	resource_gids := resource.GetWriters()
	return groupsOverlap(requestor_gids, resource_gids)
}

func (manager *AuthorizationManager) IsPubliclyViewable(resource Resource) bool {
	wildcard_gids := manager.getWildcardGroups()
	resource_gids := resource.GetReaders()
	return groupsOverlap(wildcard_gids, resource_gids)
}

func (manager *AuthorizationManager) EmailIsKnown(email string) bool {
	stub_credentials := &Credentials{User: (*User)(&email)}
	groups := manager.resolveGroupIDs(stub_credentials)
	if len(groups) != 0 {
		return true
	}
	return false
}
