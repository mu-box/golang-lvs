package lvs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type (
	Service struct {
		Host        string   `json:"host"`
		Port        int      `json:"port"`
		Type        string   `json:"type"`
		Scheduler   string   `json:"scheduler"`
		Persistence int      `json:"persistence"`
		Netmask     string   `json:"netmask"`
		Servers     []Server `json:"servers"`
	}
)

var (
	ServiceTypeFlag = map[string]string{
		"tcp":    "-t",
		"udp":    "-u",
		"fwmark": "-f",
		"":       "-t", // default
	}

	ServiceSchedulerFlag = map[string]string{
		"rr":    "rr",
		"wrr":   "wrr",
		"lc":    "lc",
		"wlc":   "wlc",
		"lblc":  "lblc",
		"lblcr": "lblcr",
		"dh":    "dh",
		"sh":    "sh",
		"sed":   "sed",
		"nq":    "nq",
		"":      "wlc", // default
	}

	InvalidServiceType      = errors.New("Invalid Service Type")
	InvalidServiceScheduler = errors.New("Invalid Service Scheduler")
)

func (s Service) Validate() error {
	_, ok := ServiceTypeFlag[s.Type]
	if !ok {
		return InvalidServiceType
	}
	_, ok = ServiceSchedulerFlag[s.Scheduler]
	if !ok {
		return InvalidServiceScheduler
	}
	for _, server := range s.Servers {
		err := server.Validate()
		if err != nil {
			return err
		}
		// follow ipvsadm rules
		if server.Forwarder != "m" && (s.Port != server.Port) {
			return InvalidServerPort
		}
	}
	return nil
}

func (s Service) FindServer(host string, port int) *Server {
	for i := range s.Servers {
		if s.Servers[i].Host == host && s.Servers[i].Port == port {
			return &s.Servers[i]
		}
	}
	return nil
}

func (s *Service) AddServer(server Server) error {
	err := server.Validate()
	if err != nil {
		return err
	}
	if server.Forwarder != "m" && (s.Port != server.Port) {
		return InvalidServerPort
	}
	if s.FindServer(server.Host, server.Port) != nil {
		return nil
	}
	err = backend("ipvsadm", append([]string{"-a", ServiceTypeFlag[s.Type], s.getHostPort(), "-r"}, strings.Split(server.String(), " ")...)...)
	if err != nil {
		return err
	}

	s.Servers = append(s.Servers, server)
	return nil
}

func (s *Service) EditServer(server Server) error {
	err := server.Validate()
	if err != nil {
		return err
	}
	if server.Forwarder != "m" && (s.Port != server.Port) {
		return InvalidServerPort
	}

	err = backend("ipvsadm", append([]string{"-e", ServiceTypeFlag[s.Type], s.getHostPort(), "-r"}, strings.Split(server.String(), " ")...)...)
	if err != nil {
		return err
	}

	for i := range s.Servers {
		if s.Servers[i].Host == server.Host && s.Servers[i].Port == server.Port {
			s.Servers = append(s.Servers[:i], append([]Server{server}, s.Servers[i+1:]...)...)
			break
		}
	}
	return nil
}

func (s *Service) RemoveServer(host string, port int) error {
	err := backend("ipvsadm", "-d", ServiceTypeFlag[s.Type], s.getHostPort(), "-r", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}

	for i := range s.Servers {
		if s.Servers[i].Host == host && s.Servers[i].Port == port {
			s.Servers = append(s.Servers[:i], s.Servers[i+1:]...)
			break
		}
	}
	return nil
}

func (s *Service) FromJson(bytes []byte) error {
	return json.Unmarshal(bytes, s)
}

func (s Service) ToJson() ([]byte, error) {
	return json.Marshal(s)
}

func (s Service) getNetmask() []string {
	if s.Netmask != "" {
		return []string{"-M", s.Netmask}
	} else {
		return []string{}
	}
}

func (s Service) getPersistence() []string {
	if s.Persistence != 0 {
		return []string{"-p", fmt.Sprintf("%d", s.Persistence)}
	} else {
		return []string{}
	}
}

func (s Service) getHostPort() string {
	if s.Port == 0 {
		return s.Host
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s Service) String() string {
	a := make([]string, 0, 0)
	a = append(a, fmt.Sprintf("-A %s %s -s %s %s %s\n",
		ServiceTypeFlag[s.Type], s.getHostPort(),
		ServiceSchedulerFlag[s.Scheduler], strings.Join(s.getPersistence(), " "), strings.Join(s.getNetmask(), " ")))
	for i := range s.Servers {
		a = append(a, fmt.Sprintf("-a %s %s:%d -r %s\n",
			ServiceTypeFlag[s.Type], s.Host, s.Port,
			s.Servers[i].String()))
	}
	return strings.Join(a, "")
}

func (s Service) Add() error {
	return backend("ipvsadm", append([]string{"-A", ServiceTypeFlag[s.Type], s.getHostPort(), "-s", ServiceSchedulerFlag[s.Scheduler]}, append(s.getPersistence(), s.getNetmask()...)...)...)
}

func (s Service) Remove() error {
	return backend("ipvsadm", "-D", ServiceTypeFlag[s.Type], s.getHostPort())
}

func (s Service) Zero() error {
	return backend("ipvsadm", "-Z", ServiceTypeFlag[s.Type], s.getHostPort())
}

func parseService(serviceString string) Service {
	service := Service{
		Scheduler:   "wlc",
		Type:        "tcp",
		Persistence: 300,
	}
	var err error
	exploded := strings.Split(serviceString, " ")
	for i := range exploded {
		switch exploded[i] {
		case "-t", "--tcp-service":
			service.Type = "tcp"
			service.Host, service.Port = parseHostPort(exploded[i+1])
		case "-u", "--udp-service":
			service.Type = "udp"
			service.Host, service.Port = parseHostPort(exploded[i+1])
		case "-f", "--fwmark-service":
			service.Type = "fwmark"
			service.Host, service.Port = parseHostPort(exploded[i+1])
		case "-s", "--scheduler":
			service.Scheduler = exploded[i+1]
		case "-p", "--persistent":
			service.Persistence, err = strconv.Atoi(exploded[i+1])
			if err != nil {
				service.Persistence = 300
			}
		case "-M", "--netmask":
			service.Netmask = exploded[i+1]
		}
	}
	return service
}
