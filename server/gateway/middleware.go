// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

var (
	businessCategoryOid        = asn1.ObjectIdentifier{2, 5, 4, 15}
	businessCategoryProduction = "production"
)

func (h handlers) authDevice(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		cert := req.TLS.PeerCertificates[0]
		uuid := cert.Subject.CommonName
		ctx := req.Context()
		log := CtxGetLog(ctx).With("device", uuid)
		ctx = CtxWithLog(ctx, log)

		isProd := getBusinessCategory(cert.Subject) == businessCategoryProduction
		pub, err := pubkey(cert)
		if err != nil {
			return c.String(http.StatusForbidden, fmt.Sprintf("unable to extract device's public key: %s", err))
		}

		device, err := h.storage.DeviceGet(uuid)

		if err != nil {
			log.Error("Unable to query for device", "error", err)
			return c.String(http.StatusBadGateway, err.Error())
		} else if device == nil {
			device, err = h.storage.DeviceCreate(cert.Subject.CommonName, pub, isProd)
			if err != nil {
				log.Error("Unable to create device", "error", err)
				return c.String(http.StatusBadGateway, err.Error())
			}
			log.Info("Created device")
		} else if device.Deleted {
			return c.String(http.StatusForbidden, fmt.Sprintf("Device(%s) has been deleted", uuid))
		} else if pub != device.PubKey {
			/*if err := device.RotatePubKey(pub); err != nil {
				return c.String(http.StatusForbidden, err.Error())
			}*/
			panic("TODO ROTATE KEY")
		}

		ctx = CtxWithDevice(ctx, device)
		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}

func pubkey(cert *x509.Certificate) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

// Golang crypto/x509/pkix package doesn't parse a dozen of standard attributes
func getBusinessCategory(subject pkix.Name) string {
	for _, atv := range subject.Names {
		if businessCategoryOid.Equal(atv.Type) {
			return atv.Value.(string)
		}
	}
	return ""
}
