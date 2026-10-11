package docker

const (
	LabelManaged      = "ebox.managed"
	LabelResourceType = "ebox.resource-type"
	LabelResourceID   = "ebox.resource-id"
	LabelDeploymentID = "ebox.deployment-id"
	LabelName         = "ebox.name"

	ResourceApplication = "application"
	ResourceDatabase    = "database"
	ResourceService     = "service"
	ResourceProxy       = "proxy"
)

// ManagedLabels returns the labels every managed container carries.
func ManagedLabels(resourceType, resourceID, name string) map[string]string {
	return map[string]string{
		LabelManaged:      "true",
		LabelResourceType: resourceType,
		LabelResourceID:   resourceID,
		LabelName:         name,
	}
}

// ResourceFilter builds a Docker filter selecting managed containers of one
// resource. Docker expects label filters as label=<key>=<value>.
func ResourceFilter(resourceType, resourceID string) Filters {
	filters := Filters{}.Add("label", LabelManaged+"=true")
	if resourceType != "" {
		filters = filters.Add("label", LabelResourceType+"="+resourceType)
	}
	if resourceID != "" {
		filters = filters.Add("label", LabelResourceID+"="+resourceID)
	}

	return filters
}
