package specs

type SpecRelationships struct {
	Package string             `json:"package"`
	Preset Preset              `json:"preset"`
	Defs []ForkTypeDefinitions `json:"defs"`
}

type ForkTypeDefinitions struct {
	Fork Fork               `json:"fork"`
	Types []TypeRelation `json:"types"`
}

type TypeRelation struct {
	SpecName string `json:"name"`
	TypeName string `json:"type_name"`
}
