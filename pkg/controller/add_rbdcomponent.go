package controller

import (
	"github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rbdcomponent.Add)
}
