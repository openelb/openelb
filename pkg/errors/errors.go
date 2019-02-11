package errors

import "fmt"

type ResourceNotEnoughError struct {
	resourceName string
}

func (r ResourceNotEnoughError) Error() string {
	return fmt.Sprintf("Not enougn %s to use", r.resourceName)
}
func NewResourceNotEnoughError(resourcename string) ResourceNotEnoughError {
	return ResourceNotEnoughError{
		resourceName: resourcename,
	}
}

type EIPNotFoundError struct {
	eip string
}

func (r EIPNotFoundError) Error() string {
	return fmt.Sprintf("EIP %s is not exsit in system", r.eip)
}
func NewEIPNotFoundError(eip string) EIPNotFoundError {
	return EIPNotFoundError{
		eip: eip,
	}
}
