// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// VolumeGroupCreate volume group create
//
// swagger:model VolumeGroupCreate
type VolumeGroupCreate struct {

	// The name of consistencyGroup at storage controller level. This is required to onboard existing volume group on the target site for DR set up; name and consistencyGroupName are mutually exclusive.
	ConsistencyGroupName string `json:"consistencyGroupName,omitempty"`

	// The name of the volume group. This field is required for creation of new volume group; name and consistencyGroupName are mutually exclusive.
	// Max Length: 120
	// Pattern: ^[\s]*[A-Za-z0-9:_.\-][A-Za-z0-9\s:_.\-]*$
	Name string `json:"name,omitempty"`

	// List of volume IDs,members of VolumeGroup
	// Required: true
	VolumeIDs []string `json:"volumeIDs"`
}

// Validate validates this volume group create
func (m *VolumeGroupCreate) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVolumeIDs(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *VolumeGroupCreate) validateName(formats strfmt.Registry) error {
	if swag.IsZero(m.Name) { // not required
		return nil
	}

	if err := validate.MaxLength("name", "body", m.Name, 120); err != nil {
		return err
	}

	if err := validate.Pattern("name", "body", m.Name, `^[\s]*[A-Za-z0-9:_.\-][A-Za-z0-9\s:_.\-]*$`); err != nil {
		return err
	}

	return nil
}

func (m *VolumeGroupCreate) validateVolumeIDs(formats strfmt.Registry) error {

	if err := validate.Required("volumeIDs", "body", m.VolumeIDs); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this volume group create based on context it is used
func (m *VolumeGroupCreate) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VolumeGroupCreate) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VolumeGroupCreate) UnmarshalBinary(b []byte) error {
	var res VolumeGroupCreate
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
