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

func IsResourceNotFound(err error) bool {
	_, ok := err.(EIPNotFoundError)
	return ok
}

func IsEIPNotEnough(err error) bool {
	_, ok := err.(ResourceNotEnoughError)
	return ok
}

type DataStoreEIPDuplicateError struct {
	CIDR string
}

func (e DataStoreEIPDuplicateError) Error() string {
	return fmt.Sprintf("%s is duplicated because it is a subnet of current pool", e.CIDR)
}

type DataStoreEIPNotExist struct {
	CIDR string
}

func (e DataStoreEIPNotExist) Error() string {
	return fmt.Sprintf("%s is not in current pool", e.CIDR)
}

type DataStoreEIPIsUsedError struct {
	CIDR string
}

func (e DataStoreEIPIsUsedError) Error() string {
	return fmt.Sprintf("%s is in use ", e.CIDR)
}

type DataStoreEIPIsNotUsedError struct {
	EIP string
}

func (e DataStoreEIPIsNotUsedError) Error() string {
	return fmt.Sprintf("%s is not in use ", e.EIP)
}

type DataStoreEIPIsInvalid struct {
	EIP string
}

func (e DataStoreEIPIsInvalid) Error() string {
	return fmt.Sprintf("%s is not a valid address", e.EIP)
}
