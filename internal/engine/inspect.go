package engine

import "encoding/json"

// InspectNode is one node from inspect output.
type InspectNode struct {
	Node      string
	Container string
	Kind      string
	Image     string
	Group     string
	State     string
	Status    string
	IPv4      string
	IPv6      string
}

// ParseInspect parses `containerlab inspect --format json`, grouped by lab name.
func ParseInspect(data []byte) (map[string][]InspectNode, error) {
	var raw map[string][]struct {
		LabName string `json:"lab_name"`
		Name    string `json:"name"`
		Kind    string `json:"kind"`
		Image   string `json:"image"`
		Group   string `json:"group"`
		State   string `json:"state"`
		Status  string `json:"status"`
		IPv4    string `json:"ipv4_address"`
		IPv6    string `json:"ipv6_address"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := map[string][]InspectNode{}
	for lab, nodes := range raw {
		for _, n := range nodes {
			out[lab] = append(out[lab], InspectNode{
				Node:      trimContainerPrefix(n.Name, lab),
				Container: n.Name,
				Kind:      n.Kind,
				Image:     n.Image,
				Group:     n.Group,
				State:     n.State,
				Status:    n.Status,
				IPv4:      n.IPv4,
				IPv6:      n.IPv6,
			})
		}
	}
	return out, nil
}

// InspectInterface is one interface from inspect interfaces output.
type InspectInterface struct {
	Name  string
	Alias string
	MAC   string
	MTU   int
	State string
}

// ParseInspectInterfaces parses `containerlab inspect interfaces --format json`.
func ParseInspectInterfaces(data []byte) (map[string][]InspectInterface, error) {
	var raw []struct {
		Name       string `json:"name"`
		Interfaces []struct {
			Name  string `json:"name"`
			Alias string `json:"alias"`
			MAC   string `json:"mac"`
			MTU   int    `json:"mtu"`
			State string `json:"state"`
		} `json:"interfaces"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := map[string][]InspectInterface{}
	for _, node := range raw {
		for _, i := range node.Interfaces {
			out[node.Name] = append(out[node.Name], InspectInterface{
				Name: i.Name, Alias: i.Alias, MAC: i.MAC, MTU: i.MTU, State: i.State,
			})
		}
	}
	return out, nil
}

func trimContainerPrefix(container, lab string) string {
	prefix := "clab-" + lab + "-"
	if len(container) > len(prefix) && container[:len(prefix)] == prefix {
		return container[len(prefix):]
	}
	return container
}
