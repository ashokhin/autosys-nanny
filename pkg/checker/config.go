package checker

import "github.com/ashokhin/autosys-nanny/pkg/mailer"

type CheckerConfig struct {
	Services []*Service     `yaml:"services_list"`
	Mailer   *mailer.Mailer `yaml:"general"`
	to       []string
}
