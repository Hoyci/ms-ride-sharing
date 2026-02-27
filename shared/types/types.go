package types

import userpb "ms-ride-sharing/shared/proto/v1/user"

type UserType string

const (
	DRIVER    UserType = "DRIVER"
	PASSENGER UserType = "PASSENGER"
)

func MapUserTypeDomainToProto(t UserType) userpb.UserType {
	switch t {
	case DRIVER:
		return userpb.UserType_DRIVER
	case PASSENGER:
		return userpb.UserType_RIDER
	default:
		return userpb.UserType_UNSPECIFIED
	}
}

func MapProtoToUserTypeDomain(t userpb.UserType) UserType {
	switch t {
	case userpb.UserType_DRIVER:
		return DRIVER
	case userpb.UserType_RIDER:
		return PASSENGER
	default:
		return PASSENGER
	}
}
