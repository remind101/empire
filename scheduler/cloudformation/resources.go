package cloudformation

type PortMappingProperties struct {
	ContainerPort interface{} `json:",omitempty"`
	HostPort      interface{} `json:",omitempty"`
}

type ContainerDefinitionProperties struct {
	Command          interface{}              `json:",omitempty"`
	Cpu              interface{}              `json:",omitempty"`
	DockerLabels     map[string]interface{}   `json:",omitempty"`
	Environment      interface{}              `json:",omitempty"`
	Essential        interface{}              `json:",omitempty"`
	Image            interface{}              `json:",omitempty"`
	Memory           interface{}              `json:",omitempty"`
	Name             interface{}              `json:",omitempty"`
	PortMappings     []*PortMappingProperties `json:",omitempty"`
	Ulimits          interface{}              `json:",omitempty"`
	LogConfiguration interface{}              `json:",omitempty"`
}

type TaskDefinitionProperties struct {
	ContainerDefinitions []*ContainerDefinitionProperties `json:",omitempty"`
	Volumes              []interface{}
}

type CustomTaskDefinitionProperties struct {
	ContainerDefinitions []*ContainerDefinitionProperties `json:",omitempty"`
	Family               interface{}                      `json:",omitempty"`
	ServiceToken         interface{}                      `json:",omitempty"`
	Volumes              []interface{}
}
